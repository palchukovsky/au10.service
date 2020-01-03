package au10

// MessageID is message ID.
type MessageID uint64

// MessageKind is message kind ID.
type MessageKind uint32

const (
	// MessageKindText is ID for message kind "text".
	MessageKindText MessageKind = 0
)

// MessageDeclaration gets short information about message.
type MessageDeclaration struct {
	Kind MessageKind
	Size uint64
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
