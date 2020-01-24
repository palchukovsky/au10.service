package au10

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Shopify/sarama"
)

////////////////////////////////////////////////////////////////////////////////

// StreamReader describes a data stream reading client.
type StreamReader interface {
	Close()
	NewSubscription(
		handle func(interface{}),
		errChan chan<- error) (StreamSubscription, error)
}

// StreamSubscription describes subscription to a data from stream.
type StreamSubscription interface {
	Close()
}

type streamSubscription struct {
	stream  *streamReader
	handle  func(interface{})
	errChan chan<- error
}

func (subscription *streamSubscription) Close() {
	if subscription.stream == nil {
		// never subscribed
		return
	}
	if err := subscription.stream.request(subscription); err != nil {
		subscription.stream.service.Log().Error(
			`Failed to close stream subscription: "%s".`, err)
	}
}

type streamHandler struct {
	stream *streamReader
}

func (*streamHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (*streamHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (handler *streamHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {

	for message := range claim.Messages() {
		convertedMessage, err := handler.stream.convertMessage(message)
		if err != nil {
			return fmt.Errorf(`failed to convert message from stream "%s": "%s"`,
				handler.stream.getTopic(), err)
		}
		handler.stream.messagesChan <- convertedMessage
		session.MarkMessage(message, "")
	}
	return nil
}

type streamReader struct {
	service Service

	topics []string

	convertMessage func(*sarama.ConsumerMessage) (interface{}, error)

	requestsChan  chan *streamSubscription
	responsesChan chan error
	messagesChan  chan interface{}
	errChan       chan error

	subscriptions map[*streamSubscription]interface{}

	cancelConsume context.CancelFunc
	stopBarrier   sync.WaitGroup
	consumer      sarama.ConsumerGroup
}

func (*factory) NewStreamReader(
	topics []string,
	convertMessage func(*sarama.ConsumerMessage) (interface{}, error),
	service Service) StreamReader {
	result := &streamReader{
		service:        service,
		topics:         topics,
		convertMessage: convertMessage,
		requestsChan:   make(chan *streamSubscription),
		responsesChan:  make(chan error),
		messagesChan:   make(chan interface{}, 1),
		errChan:        make(chan error, 1),
		subscriptions:  map[*streamSubscription]interface{}{}}
	go result.serve()
	return result
}

func (stream *streamReader) Close() {
	if err := stream.request(nil); err != nil {
		stream.service.Log().Error(`Failed to close stream reader: "%s".`, err)
	}
	if len(stream.subscriptions) != 0 {
		stream.service.Log().Error(
			`Not all (%d) subscribes of stream reading "%s" closed.`,
			len(stream.subscriptions), stream.getTopic())
	}
	close(stream.errChan)
	close(stream.messagesChan)
	close(stream.responsesChan)
	close(stream.requestsChan)
}

func (stream *streamReader) NewSubscription(
	handle func(interface{}),
	errChan chan<- error) (StreamSubscription, error) {
	subscription := &streamSubscription{handle: handle, errChan: errChan}
	err := stream.request(subscription)
	if err != nil {
		subscription.Close()
		return nil, err
	}
	return subscription, nil
}

func (stream *streamReader) serve() {
	for {
		select {
		case subscription := <-stream.requestsChan:
			if subscription == nil {
				stream.close()
				stream.responsesChan <- nil
				return
			}
			if subscription.stream == nil {
				err := stream.subscribe(subscription)
				if err == nil {
					subscription.stream = stream
				}
				stream.responsesChan <- err
			} else {
				stream.unsubscribe(subscription)
				stream.responsesChan <- nil
			}
		case message := <-stream.messagesChan:
			for subscription := range stream.subscriptions {
				subscription.handle(message)
			}
		case err := <-stream.errChan:
			for subscription := range stream.subscriptions {
				subscription.errChan <- err
			}
		}
	}
}

func (stream *streamReader) close() {
	if stream.consumer == nil {
		return
	}
	stream.stop()
}

func (stream *streamReader) subscribe(subscription *streamSubscription) error {
	if stream.consumer == nil {
		if err := stream.start(); err != nil {
			return fmt.Errorf(
				`failed to start steam reading "%s" to subscribe: "%s"`,
				stream.getTopic(), err)
		}
	}
	stream.subscriptions[subscription] = nil
	return nil
}

func (stream *streamReader) unsubscribe(subscription *streamSubscription) {
	delete(stream.subscriptions, subscription)
	if len(stream.subscriptions) == 0 {
		stream.stop()
	}
}

func (stream *streamReader) request(subscription *streamSubscription) error {
	stream.requestsChan <- subscription
	response := <-stream.responsesChan
	return response
}

func (stream *streamReader) start() error {
	var err error
	stream.consumer, err = stream.service.GetFactory().NewSaramaConsumer(
		stream.service)
	if err != nil {
		return fmt.Errorf(`failed to open stream reading "%s": "%s"`,
			stream.getTopic(), err)
	}

	var ctx context.Context
	ctx, stream.cancelConsume = context.WithCancel(context.Background())
	stream.stopBarrier = sync.WaitGroup{}
	stream.stopBarrier.Add(1)
	go func() {
		handler := &streamHandler{stream: stream}
		for {
			err := stream.consumer.Consume(ctx, stream.topics, handler)
			if err == nil {
				if err = ctx.Err(); err == nil {
					continue
				}
			}
			if err != sarama.ErrClosedConsumerGroup && err != context.Canceled {
				stream.errChan <- fmt.Errorf(
					`failed to consume message from stream "%s": "%s"`,
					stream.getTopic(), err)
			} else {
				stream.errChan <- nil
			}
			break
		}
		stream.stopBarrier.Done()
	}()

	stream.service.Log().Debug(`Stream reading "%s" opened.`, stream.getTopic())
	return nil
}

func (stream *streamReader) stop() {
	stream.cancelConsume()
	stream.stopBarrier.Wait()
	if err := stream.consumer.Close(); err != nil {
		stream.service.Log().Error(`Failed to close stream reading "%s": "%s".`,
			stream.getTopic(), err)
	} else {
		stream.service.Log().Debug(`Stream reading "%s" closed.`, stream.getTopic())
	}
	stream.consumer = nil
}

func (stream *streamReader) getTopic() string {
	return strings.Join(stream.topics, ", ")
}
