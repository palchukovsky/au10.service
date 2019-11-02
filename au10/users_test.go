package au10_test

import (
	"errors"
	"testing"

	"bitbucket.org/au10/service/au10"
	"bitbucket.org/au10/service/mock/au10"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_Users_FindUser(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	users, err := au10.CreateFactory().CreateUsers(au10.CreateFactory())
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

func Test_Au10_Users_FindSession(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	users, err := au10.CreateFactory().CreateUsers(au10.CreateFactory())
	assert.NoError(err)
	defer users.Close()

	u, err := users.FindSession("domain_root")
	assert.Nil(u)
	assert.NoError(err)
}

func Test_Au10_Users_All(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	users, err := au10.CreateFactory().CreateUsers(au10.CreateFactory())
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
	factory.EXPECT().CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(mock_au10.NewMockUser(mock), nil)
	factory.EXPECT().CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("test error"))

	users, err := au10.CreateFactory().CreateUsers(factory)
	assert.Nil(users)
	assert.EqualError(err, "test error")
}
