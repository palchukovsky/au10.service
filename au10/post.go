package au10

import "time"

// PostID is a post ID.
type PostID uint32

// PostKind is post kind ID.
type PostKind uint32

const (
	// PostKindVocal is ID for post with kind "vocal".
	PostKindVocal PostKind = 0
)

// Post describes existent post.
type Post interface {
	Member
	// GetID returns post ID.
	GetID() PostID
	// GetKind returns post kind code.
	GetKind() PostKind
	// GetTime returns time when post was published.
	GetTime() time.Time
	// GetMessages returns list of post messages.
	GetMessages() []Message
}

// CreatePost creates new post object.
func CreatePost(id PostID, kind PostKind) Post {
	return &post{
		membership: CreateMembership("", ""),
		id:         id,
		kind:       kind}
}

type post struct {
	membership Membership
	id         PostID
	kind       PostKind
	time       time.Time
	messages   []Message
}

func (post *post) GetMembership() Membership { return post.membership }
func (post *post) GetID() PostID             { return post.id }
func (post *post) GetKind() PostKind         { return post.kind }
func (post *post) GetTime() time.Time        { return post.time }
func (post *post) GetMessages() []Message    { return post.messages }
