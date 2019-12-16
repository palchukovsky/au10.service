package au10

// Posts describes post collection service.
type Posts interface {
	Member

	// Close closes the service.
	Close()

	// Add adds new post.
	Add(User, PostData) error
}

func (*factory) CreatePosts(service Service) Posts {
	return &posts{
		service:    service,
		membership: CreateMembership("", "")}
}

type posts struct {
	service    Service
	membership Membership
}

func (*posts) Close() {}

func (posts *posts) GetMembership() Membership { return posts.membership }

func (*posts) Add(user User, post PostData) error {
	return nil
}
