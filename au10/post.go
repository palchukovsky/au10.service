package au10

// Post describes existent post.
type Post interface {
	Member
}

// CreatePost creates new post object.
func CreatePost() Post {
	return &post{
		membership: CreateMembership("", "")}
}

// PostData describes abastart post post data.
type PostData interface {
	SetText(string)
	GetText() string
}

// CreatePostData creates new post data object.
func CreatePostData() PostData {
	return &postData{}
}

////////////////////////////////////////////////////////////////////////////////

type post struct {
	membership Membership
}

func (post *post) GetMembership() Membership { return post.membership }

////////////////////////////////////////////////////////////////////////////////

type postData struct {
	text string
}

func (data *postData) GetText() string     { return data.text }
func (data *postData) SetText(text string) { data.text = text }

////////////////////////////////////////////////////////////////////////////////
