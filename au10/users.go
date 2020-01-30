package au10

import (
	"fmt"
	"strings"
)

// Users provides a user database interface.
type Users interface {
	// Close closes database.
	Close()
	// Auth verifies user credentials and creates token if credentials
	// are correct. Returns nil instead token if credentials are wrong.
	Auth(login string) (User, *string, error)
	// Find tries to find a user by the login. Returns nil if the user isn't
	// found.
	FindUser(login string) (User, error)
	// Returns user by ID.
	GetUser(UserID) (User, error)
	// Find tries to find a user by session token. Returns nil if the user isn't
	// found.
	FindSession(token string) (User, error)
	// GetAll returns all users from the database.
	GetAll() []User
}

// NewUsers creates new users service instance.
func NewUsers(service Service) (Users, error) {
	result := &users{service: service, users: map[string]User{}}
	factory := result.service.GetFactory()
	var err error
	result.newUser(123, "root", "", "", []Rights{NewRights("*", "*")}, factory, &err)
	result.newUser(234, "domain_root", "x-company", "", []Rights{NewRights("x-company", "*")}, factory, &err)
	result.newUser(345, "domain_admin", "x-company", "admins",
		[]Rights{NewRights("x-company", "users"), NewRights("x-company", "admins")}, factory, &err)
	result.newUser(456, "user1", "x-company", "users", []Rights{NewRights("x-company", "users")}, factory, &err)
	result.newUser(567, "user2", "x-company", "users", []Rights{NewRights("x-company", "users")}, factory, &err)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type users struct {
	service Service
	users   map[string]User
}

func (users *users) newUser(
	id UserID,
	login, domain, group string,
	rights []Rights,
	factory Factory,
	err *error) {
	if *err != nil {
		return
	}
	var user User
	user, *err = factory.NewUser(
		id, login, NewMembership(domain, group), rights, users.service)
	if *err != nil {
		return
	}
	users.users[login] = user
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

func (users *users) GetUser(id UserID) (User, error) {
	for _, u := range users.users {
		if u.GetID() == id {
			return u, nil
		}
	}
	return nil, fmt.Errorf("failed to find user with ID %d", id)
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
