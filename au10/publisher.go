package au10

import (
	"encoding/json"
	"fmt"

	"github.com/Shopify/sarama"
)

const publisherPostsStreamTopic = "pub-posts"
const publisherMessagesStreamTopic = "pub-messages"

////////////////////////////////////////////////////////////////////////////////

// Publisher is a service that publishes new posts.
type Publisher interface {
	Member
	// Close closes the service.
	Close()
	// AddVocal accepts new vocal for publications. The vocal will be published
	// when the last message will be uploaded.
	AddVocal(*VocalDeclaration) (Vocal, error)
	// AppendMessage accepts part of new message to publish. The messages will be
	// published when the last message part will be added.
	AppendMessage(MessageID, PostID, User, []byte) error
}

type publisher struct {
	membership     Membership
	postsStream    StreamSyncWriter
	messagesStream StreamAsyncWriter
	log            Log
}

// NewPublisher creates new post publisher instance.
func NewPublisher(service Service) (Publisher, error) {
	postsStream, err := service.GetFactory().NewStreamSyncWriter(
		publisherPostsStreamTopic, service)
	if err != nil {
		return nil, err
	}
	result := &publisher{
		postsStream: postsStream,
		membership:  NewMembership("", ""),
		log:         service.Log()}
	result.messagesStream, err = service.GetFactory().NewStreamAsyncWriter(
		publisherMessagesStreamTopic, service)
	if err != nil {
		postsStream.Close()
		return nil, err
	}
	return result, nil
}

func (publisher *publisher) Close() {
	publisher.messagesStream.Close()
	publisher.postsStream.Close()
}

func (publisher *publisher) GetMembership() Membership {
	return publisher.membership
}

// PublisherMessageData is a serializable record for a stream that informs about
// message data.
type PublisherMessageData struct {
	Post   PostID    `json:"p"`
	ID     MessageID `json:"m"`
	Author UserID    `json:"a"`
	Data   []byte    `json:"d"`
}

func (publisher *publisher) AddVocal(
	vocal *VocalDeclaration) (Vocal, error) {
	id, time, err := publisher.postsStream.Push(vocal)
	if err != nil {
		return nil, err
	}
	return newNewVocal(PostID(id), time, vocal), nil
}

func (publisher *publisher) AppendMessage(
	id MessageID, post PostID, user User, data []byte) error {
	return publisher.messagesStream.PushAsync(&PublisherMessageData{
		Post: post, ID: id, Author: user.GetID(), Data: data})
}

////////////////////////////////////////////////////////////////////////////////

// PublishPostsStream represents the interface to read new post assigned
// for publication.
type PublishPostsStream interface {
	Subscription
	// GetRecordsChan resturns incoming records channel.
	GetRecordsChan() <-chan Vocal
}

// PublishMessagesStream represents the interface to read new messages
// assigned for publication.
type PublishMessagesStream interface {
	Subscription
	// GetRecordsChan resturns incoming records channel.
	GetRecordsChan() <-chan *PublisherMessageData
}

type publishPostsStreamSingleton struct {
	subscription
	reader      StreamReader
	stream      StreamSubscription
	recordsChan chan Vocal
}

// NewPublishPostsStreamSingleton creates new publish queue stream instance
// singleton (allowed only one instance for a proccess).
func NewPublishPostsStreamSingleton(
	service Service) (PublishPostsStream, error) {

	result := &publishPostsStreamSingleton{
		subscription: newSubscription(),
		reader: service.GetFactory().NewStreamReader(
			[]string{publisherPostsStreamTopic},
			func(source *sarama.ConsumerMessage) (interface{}, error) {
				var decl VocalDeclaration
				if err := json.Unmarshal(source.Value, &decl); err != nil {
					return nil, fmt.Errorf(`failed to parse vocal declaration: "%s"`, err)
				}
				return newNewVocal(PostID(source.Offset), source.Timestamp, &decl), nil
			},
			service),
		recordsChan: make(chan Vocal, 1)}

	var err error
	result.stream, err = result.reader.NewSubscription(
		func(message interface{}) { result.recordsChan <- message.(Vocal) },
		result.errChan)
	if err != nil {
		result.Close()
		return nil, err
	}

	return result, nil
}

func (stream *publishPostsStreamSingleton) Close() {
	close(stream.recordsChan)
	if stream.stream != nil {
		stream.stream.Close()
	}
	stream.reader.Close()
	stream.subscription.close()
}

func (stream *publishPostsStreamSingleton) GetRecordsChan() <-chan Vocal {
	return stream.recordsChan
}

type publishMessagesStreamSingleton struct {
	subscription
	reader      StreamReader
	stream      StreamSubscription
	recordsChan chan *PublisherMessageData
}

// NewPublishMessagesStreamSingleton creates new publish queue stream instance
// singleton (allowed only one instance for a proccess).
func NewPublishMessagesStreamSingleton(
	service Service) (PublishMessagesStream, error) {

	result := &publishMessagesStreamSingleton{
		subscription: newSubscription(),
		reader: service.GetFactory().NewStreamReader(
			[]string{publisherMessagesStreamTopic},
			func(source *sarama.ConsumerMessage) (interface{}, error) {
				var data PublisherMessageData
				if err := json.Unmarshal(source.Value, &data); err != nil {
					return nil, fmt.Errorf(`failed to parse message data: "%s"`, err)
				}
				return &data, nil
			},
			service),
		recordsChan: make(chan *PublisherMessageData, 1)}

	var err error
	result.stream, err = result.reader.NewSubscription(
		func(message interface{}) {
			result.recordsChan <- message.(*PublisherMessageData)
		},
		result.errChan)
	if err != nil {
		result.Close()
		return nil, err
	}

	return result, nil
}

func (stream *publishMessagesStreamSingleton) Close() {
	close(stream.recordsChan)
	if stream.stream != nil {
		stream.stream.Close()
	}
	stream.reader.Close()
	stream.subscription.close()
}

func (stream *publishMessagesStreamSingleton) GetRecordsChan() <-chan *PublisherMessageData {
	return stream.recordsChan
}

////////////////////////////////////////////////////////////////////////////////
