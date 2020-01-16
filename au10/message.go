package au10

import "errors"

// MessageID is message ID.
type MessageID uint64

// MessageKind is message kind ID.
type MessageKind uint32

const (
	// MessageKindText is ID for message kind "text".
	MessageKindText MessageKind = 0
)

// MessageDeclaration describes short information about a post
// which has to be published.
type MessageDeclaration struct {
	Kind MessageKind `json:"k"`
	Size uint64      `json:"s"`
}

// Message describes post message interface.
type Message interface {
	Member
	// GetID returns message ID.
	GetID() MessageID
	// GetKind returns message kind code.
	GetKind() MessageKind
	// GetSize returns message size in bytes.
	GetSize() uint64
	// Append appends bytes to message content.
	Append([]byte) error
	// Load loads message content into the provided buffer. Decreases buffer size
	// for the last chunk, if the last chunk has the smaller size than the passed
	// buffer.
	Load(buffer *[]byte, offset uint64) error
}

// NewMessage create messag object instance.
func NewMessage(data *MessageData, post Post) Message {
	return &message{data: data, post: post}
}

// MessageData describes message record in a stream.
type MessageData struct {
	ID   MessageID   `json:"i"`
	Kind MessageKind `json:"k"`
	Size uint64      `json:"s"`
}

type message struct {
	data *MessageData
	post Post
}

func (message *message) GetMembership() Membership {
	return message.post.GetMembership()
}
func (message *message) GetID() MessageID     { return message.data.ID }
func (message *message) GetKind() MessageKind { return message.data.Kind }
func (message *message) GetSize() uint64      { return message.data.Size }
func (message *message) Append([]byte) error {
	return errors.New("not implemented")
}
func (message *message) Load(buffer *[]byte, offset uint64) error {
	return errors.New("not implemented")
}
