package au10

import "errors"

// Publisher is a service that publishes new posts.
type Publisher interface {
	Member
	// Close closes the service.
	Close()
	// PublishVocal accepts new vocal for a publications.
	PublishVocal(messages []MessageDeclaration, user User) (Vocal, error)
}

// NewPublisher creates new post publisher instance.
func NewPublisher(service Service) (Publisher, error) {
	stream, err := service.GetFactory().NewStreamWriter(publisherStream,
		service)
	if err != nil {
		return nil, err
	}
	return &publisher{
			stream:     stream,
			membership: NewMembership("", "")},
		nil
}

const publisherStream = "pubs"

type publisher struct {
	membership Membership
	stream     StreamWriter
}

func (publisher *publisher) Close() {
	publisher.stream.Close()
}

func (publisher *publisher) GetMembership() Membership {
	return publisher.membership
}

func (*publisher) PublishVocal(
	messages []MessageDeclaration, user User) (Vocal, error) {
	return nil, errors.New("not implemented")
}
