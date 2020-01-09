package au10_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"bitbucket.org/au10/service/au10"
	"bitbucket.org/au10/service/mock/au10"
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

	user, err := au10.NewFactory().NewUser(
		"test login", membership, rights)
	assert.NoError(err)
	assert.NotNil(user)

	assert.Equal("test login", user.GetLogin())

	assert.True(user.GetMembership() == membership)

	// source could not change object
	rights = append(rights, mock_au10.NewMockRights(ctrl))
	assert.Len(user.GetRights(), 2)
	assert.True(user.GetRights()[0] == rights[0])
	assert.True(user.GetRights()[1] == rights[1])

	// returned list could not change object
	rights = user.GetRights()
	prevRights := rights[0]
	rights[0] = mock_au10.NewMockRights(ctrl)
	assert.Len(user.GetRights(), 2)
	assert.True(user.GetRights()[0] != rights[0])
	assert.True(user.GetRights()[0] == prevRights)
	assert.True(user.GetRights()[1] == rights[1])
}
