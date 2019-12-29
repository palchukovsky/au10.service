package accesspoint

import (
	"context"
	"sync"
	"sync/atomic"

	proto "bitbucket.org/au10/service/accesspoint/proto"
	"bitbucket.org/au10/service/au10"
	codes "google.golang.org/grpc/codes"
)

// SubscriptionInfo stores abstract subscription information.
type SubscriptionInfo struct {
	Name                string
	NumberOfSubscribers uint32
}

////////////////////////////////////////////////////////////////////////////////

func (client *client) runLogSubscription(
	log au10.Log, stream proto.Au10_ReadLogServer) error {

	subscription, err := log.Subscribe()
	if err != nil {
		return client.CreateError(codes.Internal,
			`failed to subscribe to log: "%s"`, err)
	}
	return client.runSubscription(
		stream.Context(),
		&logSubscription{
			abstractSubscription: abstractSubscription{subscription: subscription},
			stream:               stream},
		client.service.GetLogSubscriptionInfo())
}

type logSubscription struct {
	abstractSubscription
	stream proto.Au10_ReadLogServer
}

func (subscription *logSubscription) sendNext() (bool, error) {
	select {
	case record, isOpen := <-subscription.getSubscription().GetRecordsChan():
		if !isOpen {
			return false, nil
		}
		return true, subscription.stream.Send(&proto.LogRecord{
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
	info *SubscriptionInfo) error {

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

	numberOfSubscribers := client.service.RegisterSubscriber()
	atomic.AddUint32(&info.NumberOfSubscribers, 1)

	client.LogDebug("Subscribed to %s (%d/%d).",
		info.Name, info.NumberOfSubscribers, numberOfSubscribers)

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
	atomic.AddUint32(&info.NumberOfSubscribers, ^uint32(0))
	numberOfSubscribers = client.service.UnregisterSubscriber()

	if err != nil {
		return client.CreateError(codes.Internal,
			`subscription to %s error: "%s" (%d/%d)`,
			info.Name, err, info.NumberOfSubscribers, numberOfSubscribers)
	}
	client.LogDebug("Subscription to %s canceled (%d/%d).",
		info.Name, info.NumberOfSubscribers, numberOfSubscribers)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type abstractSubscription struct {
	subscription au10.Subscription
}

////////////////////////////////////////////////////////////////////////////////
