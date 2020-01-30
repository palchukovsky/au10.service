package au10_test

import (
	"errors"
	"testing"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_Users_FindUser(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := au10.NewFactory()
	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	users, err := au10.NewUsers(service)
	assert.NoError(err)
	defer users.Close()

	u, err := users.FindUser("domain_root")
	assert.NoError(err)
	assert.NotNil(u)
	assert.Equal("domain_root", u.GetLogin())

	u, err = users.FindUser("user3")
	assert.Nil(u)
	assert.NoError(err)
}

func Test_Au10_Users_GetUser(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := au10.NewFactory()
	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	users, err := au10.NewUsers(service)
	assert.NoError(err)
	defer users.Close()

	u, err := users.GetUser(au10.UserID(345))
	assert.NoError(err)
	assert.NotNil(u)
	assert.Equal("domain_admin", u.GetLogin())
	assert.Equal(au10.UserID(345), u.GetID())

	u, err = users.GetUser(au10.UserID(0))
	assert.Nil(u)
	assert.EqualError(err, "failed to find user with ID 0")
}

func Test_Au10_Users_All(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := au10.NewFactory()
	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	users, err := au10.NewUsers(service)
	assert.NoError(err)
	defer users.Close()

	control := map[string]bool{
		"domain_root":  false,
		"domain_admin": false,
		"root":         false,
		"user1":        false,
		"user2":        false}

	for _, u := range users.GetAll() {
		set, has := control[u.GetLogin()]
		assert.True(has)
		assert.True(!set)
		control[u.GetLogin()] = true
	}

	for _, set := range control {
		assert.True(set)
	}
}

func Test_Au10_Users_Error(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)
	factory.EXPECT().NewUser(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(mock_au10.NewMockUser(mock), nil)
	factory.EXPECT().NewUser(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("test error"))

	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	users, err := au10.NewUsers(service)
	assert.Nil(users)
	assert.EqualError(err, "test error")
}

func Test_Au10_Users_Auth(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := au10.NewFactory()
	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	users, err := au10.NewUsers(service)
	assert.NoError(err)
	defer users.Close()

	var user au10.User
	var token *string
	user, token, err = users.Auth("unknown")
	assert.Nil(user)
	assert.Nil(token)
	assert.NoError(err)

	user, token, err = users.Auth("root")
	assert.NotNil(user)
	assert.Equal("root", user.GetLogin())
	assert.NotNil(token)
	assert.NoError(err)

	user, err = users.FindSession("x" + *token)
	assert.Nil(user)
	assert.NoError(err)

	user, err = users.FindSession(*token + "x")
	assert.Nil(user)
	assert.NoError(err)

	user, err = users.FindSession(*token)
	assert.NotNil(user)
	assert.NoError(err)
	assert.Equal("root", user.GetLogin())
}
