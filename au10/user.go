package au10

// UserID describes constant user ID.
type UserID uint32

// User provides the interface of a user object.
type User interface {
	Member

	// GetID returns user ID.
	GetID() UserID
	// GetLogin returns user login.
	GetLogin() string
	//GetRights returns groups with which user allows to work.
	GetRights() []Rights
}

func (*factory) NewUser(
	id UserID,
	login string,
	membership Membership,
	rights []Rights) (User, error) {

	r := &user{
		id:         id,
		login:      login,
		membership: membership,
		rights:     append([]Rights(nil), rights...)}
	return r, nil
}

type user struct {
	id         UserID
	login      string
	membership Membership
	rights     []Rights
}

func (u *user) GetID() UserID             { return u.id }
func (u *user) GetLogin() string          { return u.login }
func (u *user) GetMembership() Membership { return u.membership }

func (u *user) GetRights() []Rights {
	return append([]Rights(nil), u.rights...)
}
