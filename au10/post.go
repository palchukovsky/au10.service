package au10

import "time"

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

// CreatePost creates new post object.
func CreatePost(id PostID, location *GeoPoint) Post {
	return &post{
		membership: CreateMembership("", ""),
		id:         id,
		location:   location}
}

type post struct {
	membership Membership
	id         PostID
	time       time.Time
	messages   []Message
	location   *GeoPoint
}

func (post *post) GetMembership() Membership { return post.membership }
func (post *post) GetID() PostID             { return post.id }
func (post *post) GetTime() time.Time        { return post.time }
func (post *post) GetMessages() []Message    { return post.messages }
func (post *post) GetLocation() *GeoPoint    { return post.location }
