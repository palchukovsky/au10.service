package au10

import (
	"encoding/json"
	"fmt"

	"bitbucket.org/au10/service/postdb"
	"github.com/Shopify/sarama"
)

// Posts describes post database service.
type Posts interface {
	Member
	// Close closes the service.
	Close()
	// Subscribe creates a subscription to posts.
	Subscribe() (PostsSubscription, error)
	// GetQueries returns post queries client.
	GetQueries() QueryClient
}

// PostsSubscription represents subscription to posts.
type PostsSubscription interface {
	Subscription
	// GetRecordsChan returns incoming records channel.
	GetRecordsChan() <-chan Post
}

// PostNotifier represents the interface to notify about posts updates.
type PostNotifier interface {
	// PushVocal notifies push vocal in the notification queue.
	PushVocal(Vocal) error
	// Close closes the notification.
	Close()
}

////////////////////////////////////////////////////////////////////////////////

// NewPosts creates new posts service instance.
func NewPosts(service Service, db postdb.DB) Posts {
	return &posts{
		membership: NewMembership("", ""),
		stream: service.GetFactory().NewStreamReader(
			[]string{postsStreamTopic},
			func(source *sarama.ConsumerMessage) (interface{}, error) {
				return ConvertSaramaMessageIntoVocal(source)
			},
			service),
		queries: NewQueryClient(db)}
}

const postsStreamTopic = "posts"

type posts struct {
	membership Membership
	stream     StreamReader
	queries    QueryClient
}

func (posts *posts) Close() {
	posts.stream.Close()
	posts.queries.Close()
}

func (posts *posts) GetMembership() Membership { return posts.membership }
func (posts *posts) GetQueries() QueryClient   { return posts.queries }

func (posts *posts) Subscribe() (PostsSubscription, error) {

	result := &postsSubscription{
		subscription: newSubscription(),
		postsChan:    make(chan Post, 1)}

	var err error
	result.stream, err = posts.stream.NewSubscription(
		func(message interface{}) { result.postsChan <- message.(Post) },
		result.errChan)
	if err != nil {
		result.Close()
		return nil, err
	}

	return result, nil
}

////////////////////////////////////////////////////////////////////////////////

type postsSubscription struct {
	subscription
	stream    StreamSubscription
	postsChan chan Post
}

func (subscription *postsSubscription) Close() {
	if subscription.stream != nil {
		subscription.stream.Close()
	}
	close(subscription.postsChan)
	subscription.subscription.close()
}

func (subscription *postsSubscription) GetRecordsChan() <-chan Post {
	return subscription.postsChan
}

////////////////////////////////////////////////////////////////////////////////

type postNotifier struct {
	stream StreamAsyncWriter
}

// NewPostNotifier creates new post notifier instance.
func NewPostNotifier(service Service) (PostNotifier, error) {
	stream, err := service.GetFactory().NewStreamAsyncWriter(postsStreamTopic,
		service)
	if err != nil {
		return nil, err
	}
	return &postNotifier{stream: stream}, nil
}

func (notifier *postNotifier) Close() { notifier.stream.Close() }

func (notifier *postNotifier) PushVocal(vocal Vocal) error {
	return notifier.stream.PushAsync(vocal.Export())
}

////////////////////////////////////////////////////////////////////////////////

// ConvertSaramaMessageIntoVocal creates new Vocal-object from stream data.
func ConvertSaramaMessageIntoVocal(
	source *sarama.ConsumerMessage) (Vocal, error) {
	result := &vocal{post: post{membership: NewMembership("", "")}}
	if err := json.Unmarshal(source.Value, &result.data); err != nil {
		return nil, fmt.Errorf(`failed to parse vocal-record: "%s"`, err)
	}
	return result, nil
}

////////////////////////////////////////////////////////////////////////////////
