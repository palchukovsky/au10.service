package au10

import (
	"encoding/json"
	"fmt"

	"github.com/Shopify/sarama"
)

const publisherStreamTopic = "pubs"

////////////////////////////////////////////////////////////////////////////////

// Publisher is a service that publishes new posts.
type Publisher interface {
	Member
	// Close closes the service.
	Close()
	// AddVocal accepts new vocal for publications. The vocal will be published
	// when the last message will be uploaded.
	AddVocal(*VocalDeclaration) (Vocal, error)
}

type publisher struct {
	membership Membership
	stream     StreamWriterWithResult
	log        Log
}

// NewPublisher creates new post publisher instance.
func NewPublisher(service Service) (Publisher, error) {
	stream, err := service.GetFactory().NewStreamWriterWithResult(
		publisherStreamTopic, service)
	if err != nil {
		return nil, err
	}
	return &publisher{
			stream:     stream,
			membership: NewMembership("", ""),
			log:        service.Log()},
		nil
}

func (publisher *publisher) Close() {
	publisher.stream.Close()
}

func (publisher *publisher) GetMembership() Membership {
	return publisher.membership
}

func (publisher *publisher) AddVocal(
	vocal *VocalDeclaration) (Vocal, error) {
	id, time, err := publisher.stream.Push(vocal)
	if err != nil {
		return nil, err
	}
	return newVocal(PostID(id), time, vocal), nil
}

////////////////////////////////////////////////////////////////////////////////

// PublishStream represents the interface to read new post declarations
// assigned for publication.
type PublishStream interface {
	Subscription
	// GetRecordsChan resturns incoming records channel.
	GetRecordsChan() <-chan Vocal
}

type publishStreamSingleton struct {
	subscription
	reader      StreamReader
	stream      StreamSubscription
	recordsChan chan Vocal
}

// NewPublishStreamSingleton creates new publish queue stream instance
// singleton (allowed only one instance for a proccess).
func NewPublishStreamSingleton(service Service) (PublishStream, error) {

	result := &publishStreamSingleton{
		subscription: newSubscription(),
		reader: service.GetFactory().NewStreamReader(
			[]string{publisherStreamTopic},
			func(source *sarama.ConsumerMessage) (interface{}, error) {
				var decl VocalDeclaration
				if err := json.Unmarshal(source.Value, &decl); err != nil {
					return nil, fmt.Errorf(`failed to parse vocal declaration: "%s"`, err)
				}
				return newVocal(PostID(source.Offset), source.Timestamp, &decl), nil
			},
			service),
		recordsChan: make(chan Vocal, 1)}

	var err error
	result.stream, err = result.reader.NewSubscription(
		func(message interface{}) {
			result.recordsChan <- message.(Vocal)
		},
		result.errChan)
	if err != nil {
		result.Close()
		return nil, err
	}

	return result, nil
}

func (stream *publishStreamSingleton) Close() {
	close(stream.recordsChan)
	if stream.stream != nil {
		stream.stream.Close()
	}
	stream.reader.Close()
	stream.subscription.close()
}

func (stream *publishStreamSingleton) GetRecordsChan() <-chan Vocal {
	return stream.recordsChan
}

func (stream *publishStreamSingleton) GetErrChan() <-chan error {
	return stream.errChan
}

////////////////////////////////////////////////////////////////////////////////
