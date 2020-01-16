package au10

// Publisher is a service that publishes new posts.
type Publisher interface {
	Member
	// Close closes the service.
	Close()
	// PublishVocal accepts new vocal for publications.
	PublishVocal(*VocalDeclaration) (Vocal, error)
}

// NewPublisher creates new post publisher instance.
func NewPublisher(service Service) (Publisher, error) {
	stream, err := service.GetFactory().NewStreamWriterWithResult(publisherStream,
		service)
	if err != nil {
		return nil, err
	}
	return &publisher{
			stream:     stream,
			membership: NewMembership("", ""),
			log:        service.Log()},
		nil
}

const publisherStream = "pubs"

type publisher struct {
	membership Membership
	stream     StreamWriterWithResult
	log        Log
}

func (publisher *publisher) Close() {
	publisher.stream.Close()
}

func (publisher *publisher) GetMembership() Membership {
	return publisher.membership
}

func (publisher *publisher) PublishVocal(
	vocalDeclaration *VocalDeclaration) (Vocal, error) {

	id, time, err := publisher.stream.Push(vocalDeclaration)
	if err != nil {
		return nil, err
	}

	result := &vocal{
		post: post{
			membership: NewMembership("", ""),
			data: PostData{
				ID:       PostID(id),
				Time:     time.UnixNano(),
				Author:   vocalDeclaration.Author,
				Location: vocalDeclaration.Location,
				Messages: make([]*MessageData, len(vocalDeclaration.Messages))}}}
	for i, m := range vocalDeclaration.Messages {
		result.data.Messages[i] = &MessageData{
			ID:   MessageID(i),
			Kind: m.Kind,
			Size: m.Size}
	}

	return result, nil
}
