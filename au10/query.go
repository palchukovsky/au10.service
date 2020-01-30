package au10

import (
	"bitbucket.org/au10/service/postdb"
)

// QueryClient is a RPC-client to query posts.
type QueryClient interface {
	// Close closes the client.
	Close()
	// GetVocal return vocal by ID.
	GetVocal(PostID) (Vocal, error)
}

// NewQueryClient creates a new queires client.
func NewQueryClient(db postdb.DB) QueryClient {
	return &queryClient{db: db}
}

type queryClient struct {
	db postdb.DB
}

func (*queryClient) Close() {}

func (client *queryClient) GetVocal(id PostID) (Vocal, error) {
	post, err := client.db.GetPost(uint32(id))
	if err != nil {
		return nil, err
	}
	return newDBVocal(post)
}
