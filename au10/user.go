package au10

// User provides an interface of a user object.
type User interface {
	Member

	// GetLogin returns user login.
	GetLogin() string
	//GetRights returns groups with which user allows to work.
	GetRights() []Rights
}

func (*factory) NewUser(
	login string, membership Membership, rights []Rights) (User, error) {

	r := &user{
		login:      login,
		membership: membership,
		rights:     append([]Rights(nil), rights...)}
	return r, nil
}

type user struct {
	login      string
	membership Membership
	rights     []Rights
}

func (u *user) GetLogin() string          { return u.login }
func (u *user) GetMembership() Membership { return u.membership }

func (u *user) GetRights() []Rights {
	return append([]Rights(nil), u.rights...)
}
