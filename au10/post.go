package au10

import (
	"time"
)

// PostID is a post ID.
type PostID uint32

// PostDeclaration describes short information about a post
// which has to be published.
type PostDeclaration struct {
	Author   UserID                `json:"a"`
	Location *GeoPoint             `json:"l"`
	Messages []*MessageDeclaration `json:"m"`
}

// PostData describes post record in a stream.
type PostData struct {
	ID       PostID         `json:"i"`
	Time     int64          `json:"t"`
	Author   UserID         `json:"a"`
	Location *GeoPoint      `json:"l"`
	Messages []*MessageData `json:"m"`
}

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
	// GetAuthor returns author user ID.
	GetAuthor() UserID
	// Export returns post data object.
	Export() *PostData
}

// VocalDeclaration describes short information about a vocal
// which has to be published.
type VocalDeclaration struct {
	PostDeclaration
}

// Vocal describes existent post with type "vocal".
type Vocal interface {
	Post
}

func newVocal(id PostID, time time.Time, decl *VocalDeclaration) Vocal {
	result := &vocal{
		post: post{
			membership: NewMembership("", ""),
			data: PostData{
				ID:       PostID(id),
				Time:     time.UnixNano(),
				Author:   decl.Author,
				Location: decl.Location,
				Messages: make([]*MessageData, len(decl.Messages))}}}
	for i, m := range decl.Messages {
		result.data.Messages[i] = &MessageData{
			ID:   MessageID(i),
			Kind: m.Kind,
			Size: m.Size}
	}
	return result
}

type post struct {
	membership Membership
	data       PostData
}

func (post *post) GetID() PostID             { return PostID(post.data.ID) }
func (post *post) GetMembership() Membership { return post.membership }
func (post *post) GetLocation() *GeoPoint    { return post.data.Location }
func (post *post) GetAuthor() UserID         { return post.data.Author }
func (post *post) Export() *PostData         { return &post.data }
func (post *post) GetTime() time.Time {
	return time.Unix(0, post.data.Time)
}

func (post *post) GetMessages() []Message {
	result := make([]Message, len(post.data.Messages))
	for i := range post.data.Messages {
		result[i] = NewMessage(post.data.Messages[i], post)
	}
	return result
}

type vocal struct {
	post
}
