package au10

import "strings"

// Users provides a user database interface.
type Users interface {
	// Close closes database.
	Close()
	// Auth verifies user credentials and creates token if credentials
	// are correct. Returns nil instead token if credentials are wrong.
	Auth(login string) (User, *string, error)
	// Find tries to find a user by the login. Returns nil if the user isn't found.
	FindUser(login string) (User, error)
	// Find tries to find a user by session token. Returns nil if the user isn't found.
	FindSession(token string) (User, error)
	// GetAll returns all users from the database.
	GetAll() []User
}

// NewUsers creates new users service instance.
func NewUsers(factory Factory) (Users, error) {
	result := &users{users: map[string]User{}}
	var err error
	result.newUser("root", "", "", []Rights{NewRights("*", "*")}, factory, &err)
	result.newUser("domain_root", "x-company", "", []Rights{NewRights("x-company", "*")}, factory, &err)
	result.newUser("domain_admin", "x-company", "admins",
		[]Rights{NewRights("x-company", "users"), NewRights("x-company", "admins")}, factory, &err)
	result.newUser("user1", "x-company", "users", []Rights{NewRights("x-company", "users")}, factory, &err)
	result.newUser("user2", "x-company", "users", []Rights{NewRights("x-company", "users")}, factory, &err)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type users struct {
	users map[string]User
}

func (*users) Close() {}

func (users *users) Auth(login string) (User, *string, error) {
	user, has := users.users[login]
	if !has {
		return nil, nil, nil
	}
	result := "token: " + user.GetLogin()
	return user, &result, nil
}

func (users *users) FindUser(login string) (User, error) {
	result, has := users.users[login]
	if !has {
		return nil, nil
	}
	return result, nil
}

func (users *users) GetAll() []User {
	result := make([]User, len(users.users))
	i := 0
	for _, user := range users.users {
		result[i] = user
		i++
	}
	return result
}

func (users *users) FindSession(token string) (User, error) {
	if !strings.HasPrefix(token, "token: ") {
		return nil, nil
	}
	result, has := users.users[token[7:]]
	if !has {
		return nil, nil
	}
	return result, nil
}

func (users *users) newUser(
	login, domain, group string, rights []Rights, factory Factory, err *error) {
	if *err != nil {
		return
	}
	var user User
	user, *err = factory.NewUser(
		login, NewMembership(domain, group), rights)
	if *err != nil {
		return
	}
	users.users[login] = user
}
