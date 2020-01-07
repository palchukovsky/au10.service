package au10

import (
	"errors"
	"fmt"

	"github.com/Shopify/sarama"
)

// PostsStreamTopic is a name of posts stream topic.
const PostsStreamTopic = "posts"

// Posts describes post collection service.
type Posts interface {
	Member

	// Close closes the service.
	Close()

	// GetPost returns post by ID.
	GetPost(PostID) (Post, error)

	// AddVocal adds new vocal, but doesn't publishes it.
	AddVocal([]MessageDeclaration, User) (Vocal, error)

	// InitSubscriptionService initiates subscriber to read posts in the feature.
	InitSubscriptionService() error
	// Subscribe creates a subscription to posts.
	Subscribe() (PostsSubscription, error)
}

// PostsSubscription represents subscription to posts.
type PostsSubscription interface {
	Subscription
	// GetRecordsChan resturns incoming records channel.
	GetRecordsChan() <-chan Post
}

func (*factory) CreatePosts(service Service) Posts {
	return &posts{
		service:    service,
		membership: CreateMembership("", "")}
}

type posts struct {
	service    Service
	membership Membership
	reader     StreamReader
}

func (*posts) Close() {}

func (posts *posts) GetMembership() Membership { return posts.membership }

func (posts *posts) GetPost(id PostID) (Post, error) {
	return nil, fmt.Errorf("post with ID %d is nonexistent", id)
}

func (*posts) AddVocal(
	messages []MessageDeclaration, user User) (Vocal, error) {

	return nil, errors.New("not implemented")
}

func (posts *posts) InitSubscriptionService() error {
	if posts.reader != nil {
		return errors.New("posts subscription service already initiated")
	}
	posts.reader = posts.service.GetFactory().CreateStreamReader(
		[]string{PostsStreamTopic},
		func(source *sarama.ConsumerMessage) (interface{}, error) {
			return ConvertSaramaMessageIntoPost(source)
		},
		posts.service)
	return nil
}

func (posts *posts) Subscribe() (PostsSubscription, error) {
	if posts.reader == nil {
		panic("posts subscription service not initialized")
	}
	return nil, errors.New("not implemented")
}

// ConvertSaramaMessageIntoPost creates new Post-object from stream data.
func ConvertSaramaMessageIntoPost(
	source *sarama.ConsumerMessage) (Post, error) {

	return nil, fmt.Errorf("not implemented")
}
