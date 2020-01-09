package au10

import (
	"time"

	"github.com/Shopify/sarama"
)

// PostID is a post ID.
type PostID uint32

// Post describes existent post.
type Post interface {
	Member
	// GetID returns post ID.
	GetID() PostID
	// GetTime returns time when post was published.
	GetTime() time.Time
	// GetMessages returns list of post messages.
	GetMessages() []Message
	// GetLocation returns location of post creation.
	GetLocation() *GeoPoint
}

// Vocal describes existent post with type "vocal".
type Vocal interface {
	Post
}

// PostData describes post record in a stream.
type PostData struct {
	ID       PostID        `json:"i"`
	Time     int64         `json:"t"`
	Location *GeoPoint     `json:"l"`
	Messages []MessageData `json:"m"`
}

type post struct {
	membership Membership
	message    *sarama.ConsumerMessage
	data       PostData
}

func (post *post) GetID() PostID             { return PostID(post.data.ID) }
func (post *post) GetMembership() Membership { return post.membership }
func (post *post) GetLocation() *GeoPoint    { return post.data.Location }
func (post *post) GetTime() time.Time {
	return time.Unix(0, post.data.Time)
}

func (post *post) GetMessages() []Message {
	result := make([]Message, len(post.data.Messages))
	for i := range post.data.Messages {
		result[i] = NewMessage(&post.data.Messages[i], post)
	}
	return result
}
