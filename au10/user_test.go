package au10_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	"github.com/golang/mock/gomock"
)

func Test_Au10_User_Fields(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)

	membership := mock_au10.NewMockMembership(ctrl)
	rights := []au10.Rights{
		mock_au10.NewMockRights(ctrl),
		mock_au10.NewMockRights(ctrl)}

	service := mock_au10.NewMockService(ctrl)

	user, err := au10.NewFactory().NewUser(123, "test login",
		membership, rights, service)
	assert.NoError(err)
	assert.NotNil(user)

	assert.Equal(au10.UserID(123), user.GetID())
	assert.Equal("test login", user.GetLogin())
	assert.True(user.GetMembership() == membership)
	assert.Equal(rights, user.GetRights())

	log := mock_au10.NewMockLog(ctrl)
	log.EXPECT().Warn("User %d blocked by protocol mismatch.", au10.UserID(123))
	service.EXPECT().Log().Return(log)
	user.BlockByProtocolMismatch()
}
