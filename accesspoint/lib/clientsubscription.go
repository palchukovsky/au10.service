package accesspoint

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	proto "bitbucket.org/au10/service/accesspoint/proto"
	"bitbucket.org/au10/service/au10"
	codes "google.golang.org/grpc/codes"
)

// SubscriptionInfo stores abstract subscription information.
type SubscriptionInfo struct{ NumberOfSubscribers uint32 }

////////////////////////////////////////////////////////////////////////////////

func (client *client) runLogSubscription(
	log au10.Log, stream proto.Au10_ReadLogServer) error {

	subscription, err := log.Subscribe()
	if err != nil {
		return client.CreateError(codes.Internal, `failed to subscribe: "%s"`, err)
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

func (client *client) runPostsSubscription(
	posts au10.Posts, stream proto.Au10_ReadPostsServer) error {

	subscription, err := posts.Subscribe()
	if err != nil {
		return client.CreateError(codes.Internal, `failed to subscribe: "%s"`, err)
	}
	return client.runSubscription(
		stream.Context(),
		&postsSubscription{
			abstractSubscription: abstractSubscription{subscription: subscription},
			stream:               stream},
		client.service.GetPostsSubscriptionInfo())
}

type postsSubscription struct {
	abstractSubscription
	stream proto.Au10_ReadPostsServer
}

func (subscription *postsSubscription) sendNext() (bool, error) {
	select {
	case post, isOpen := <-subscription.getSubscription().GetRecordsChan():
		if !isOpen {
			return false, nil
		}
		messages := post.GetMessages()
		protoPost := &proto.Post{
			Id:       uint64(post.GetID()),
			Time:     post.GetTime().UnixNano(),
			Messages: make([]*proto.Message, len(messages))}
		var err error
		protoPost.Kind, err = subscription.convertPostKindToProto(post.GetKind())
		if err != nil {
			return false, err
		}
		for i, m := range messages {
			messageKind, err := subscription.convertMesageKindToProto(m.GetKind())
			if err != nil {
				return false, err
			}
			protoPost.Messages[i] = &proto.Message{
				Id: uint64(m.GetID()), Kind: messageKind, Size: m.GetSize()}
		}
		return true, subscription.stream.Send(protoPost)
	case err := <-subscription.getSubscription().GetErrChan():
		return false, err
	}
}

func (subscription *postsSubscription) convertPostKindToProto(
	kind au10.PostKind) (proto.Post_Kind, error) {

	switch kind {
	case au10.PostKindVocal:
		return proto.Post_VOCAL, nil
	}
	return 0, fmt.Errorf("unknown post kind %d", kind)

}
func (subscription *postsSubscription) convertMesageKindToProto(
	kind au10.MessageKind) (proto.Message_Kind, error) {

	switch kind {
	case au10.MessageKindText:
		return proto.Message_TEXT, nil
	}
	return 0, fmt.Errorf("unknown message kind %d", kind)
}

func (subscription *postsSubscription) close() {
	subscription.getSubscription().Close()
}

func (subscription *postsSubscription) getSubscription() au10.PostsSubscription {
	return subscription.subscription.(au10.PostsSubscription)
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

	client.LogDebug("Subscribed (%d/%d).",
		info.NumberOfSubscribers, numberOfSubscribers)

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
		return client.CreateError(codes.Internal, `Failed: "%s" (%d/%d)`,
			err, info.NumberOfSubscribers, numberOfSubscribers)
	}
	client.LogDebug("Canceled (%d/%d).",
		info.NumberOfSubscribers, numberOfSubscribers)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type abstractSubscription struct {
	subscription au10.Subscription
}

////////////////////////////////////////////////////////////////////////////////
