package au10

// Users provides a user database interface.
type Users interface {
	// Close closes database.
	Close()
	// Find tries to find a user by the login. Returns nil if the user isn't found.
	FindUser(login string) (User, error)
	// Find tries to find a user by session token. Returns nil if the user isn't found.
	FindSession(token string) (User, error)
	// GetAll returns all users from the database.
	GetAll() []User
}

func (*factory) CreateUsers(factory Factory) (
	Users, error) {

	result := &users{users: map[string]User{}}
	var err error
	result.createUser("root", "", "", []Rights{CreateRights("*", "*")}, factory, &err)
	result.createUser("domain_root", "x-company", "", []Rights{CreateRights("x-company", "*")}, factory, &err)
	result.createUser("domain_admin", "x-company", "admins",
		[]Rights{CreateRights("x-company", "users"), CreateRights("x-company", "admins")}, factory, &err)
	result.createUser("user1", "x-company", "users", []Rights{CreateRights("x-company", "users")}, factory, &err)
	result.createUser("user2", "x-company", "users", []Rights{CreateRights("x-company", "users")}, factory, &err)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type users struct {
	users map[string]User
}

func (*users) Close() {}

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
	return nil, nil
}

func (users *users) createUser(
	login, domain, group string, rights []Rights, factory Factory, err *error) {

	if *err != nil {
		return
	}
	var user User
	user, *err = factory.CreateUser(
		login, CreateMembership(domain, group), rights)
	if *err != nil {
		return
	}
	users.users[login] = user
}
