package accesspoint

import (
	"context"
	"sync"
	"sync/atomic"

	"bitbucket.org/au10/service/au10"
	codes "google.golang.org/grpc/codes"
)

type subscriptionInfo struct {
	name                string
	numberOfSubscribers uint32
}

////////////////////////////////////////////////////////////////////////////////

func (client *client) runLogSubscription(
	stream AccessPoint_ReadLogServer) error {

	subscription, err := client.service.au10.Log().Subscribe()
	if err != nil {
		return err
	}
	return client.runSubscription(
		stream.Context(),
		&logSubscription{
			abstractSubscription: abstractSubscription{subscription: subscription},
			stream:               stream},
		&client.service.logSubscriptionInfo)
}

type logSubscription struct {
	abstractSubscription
	stream AccessPoint_ReadLogServer
}

func (subscription *logSubscription) sendNext() (bool, error) {
	select {
	case record, isOpen := <-subscription.getSubscription().GetRecordsChan():
		if !isOpen {
			return false, nil
		}
		return true, subscription.stream.Send(&LogRecord{
			SeqNum:   record.GetSequenceNumber(),
			Time:     record.GetTime().UnixNano(),
			Text:     record.GetText(),
			Severity: record.GetSeverity(),
			NodeType: record.GetNodeType(),
			NodeName: record.GetNodeName()})
	case err := <-subscription.getSubscription().GetErrChan():
		return false, err
	}
}

func (subscription *logSubscription) close() {
	subscription.getSubscription().Close()
}

func (subscription *logSubscription) getSubscription() au10.LogSubscription {
	return subscription.subscription.(au10.LogSubscription)
}

////////////////////////////////////////////////////////////////////////////////

type subscription interface {
	close()
	sendNext() (bool, error)
}

func (client *client) runSubscription(
	ctx context.Context,
	subscription subscription,
	info *subscriptionInfo) error {

	errChan := make(chan error, 1)
	var stopBarrier sync.WaitGroup
	stopBarrier.Add(1)
	go func() {
		for {
			isOpened, err := subscription.sendNext()
			if err != nil || !isOpened {
				errChan <- err
				break
			}
		}
		stopBarrier.Done()
	}()

	atomic.AddUint32(&client.service.numberOfSubscribers, 1)
	atomic.AddUint32(&info.numberOfSubscribers, 1)

	client.logInfo("Subscribed to %s (%d/%d subscriptions).",
		info.name, info.numberOfSubscribers, client.service.numberOfSubscribers)

	var err error
	select {
	case err = <-errChan:
		break
	case <-ctx.Done():
		break
	}

	subscription.close()
	stopBarrier.Wait()
	close(errChan)
	atomic.AddUint32(&info.numberOfSubscribers, ^uint32(0))
	atomic.AddUint32(&client.service.numberOfSubscribers, ^uint32(0))

	if err != nil {
		return client.createError(codes.Internal,
			`subscription to %s error: %s" (%d/%d subscribers)`,
			info.name, err, info.numberOfSubscribers,
			client.service.numberOfSubscribers)
	}
	client.logInfo("Subscription to %s canceled (%d/%d subscribers).",
		info.name, info.numberOfSubscribers, client.service.numberOfSubscribers)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type abstractSubscription struct {
	subscription au10.Subscription
}

////////////////////////////////////////////////////////////////////////////////
