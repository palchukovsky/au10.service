package au10

import (
	"encoding/json"
	"fmt"

	"github.com/Shopify/sarama"
)

// Posts describes post database service.
type Posts interface {
	Member

	// Close closes the service.
	Close()

	// GetPost returns post by ID.
	GetPost(PostID) (Post, error)

	// Subscribe creates a subscription to posts.
	Subscribe() (PostsSubscription, error)
}

// PostsSubscription represents subscription to posts.
type PostsSubscription interface {
	Subscription
	// GetRecordsChan resturns incoming records channel.
	GetRecordsChan() <-chan Post
}

////////////////////////////////////////////////////////////////////////////////

// NewPosts creates new posts service instance.
func NewPosts(service Service) Posts {
	return &posts{
		membership: NewMembership("", ""),
		stream: service.GetFactory().NewStreamReader(
			[]string{postsStreamTopic},
			func(source *sarama.ConsumerMessage) (interface{}, error) {
				return ConvertSaramaMessageIntoPost(source)
			},
			service)}
}

const postsStreamTopic = "posts"

type posts struct {
	membership Membership
	stream     StreamReader
}

func (posts *posts) Close() {
	posts.stream.Close()
}

func (posts *posts) GetMembership() Membership { return posts.membership }

func (posts *posts) GetPost(id PostID) (Post, error) {
	return nil, fmt.Errorf("post with ID %d is nonexistent", id)
}

func (posts *posts) Subscribe() (PostsSubscription, error) {
	return newPostsSubscription(posts, posts.stream)
}

////////////////////////////////////////////////////////////////////////////////

type postsSubscription struct {
	subscription
	stream    StreamSubscription
	postsChan chan Post
}

func newPostsSubscription(
	posts *posts,
	reader StreamReader) (PostsSubscription, error) {

	result := &postsSubscription{
		subscription: newSubscription(),
		postsChan:    make(chan Post, 1)}

	var err error
	result.stream, err = reader.NewSubscription(
		func(message interface{}) { result.postsChan <- message.(Post) },
		result.errChan)
	if err != nil {
		result.Close()
		return nil, err
	}
	return result, nil
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

func (subscription *postsSubscription) GetErrChan() <-chan error {
	return subscription.errChan
}

////////////////////////////////////////////////////////////////////////////////

// ConvertSaramaMessageIntoPost creates new Post-object from stream data.
func ConvertSaramaMessageIntoPost(
	source *sarama.ConsumerMessage) (Post, error) {
	result := &post{membership: NewMembership("", "")}
	if err := json.Unmarshal(source.Value, &result.data); err != nil {
		return nil, fmt.Errorf(`failed to parse post-record: "%s"`, err)
	}
	return result, nil
}

////////////////////////////////////////////////////////////////////////////////
