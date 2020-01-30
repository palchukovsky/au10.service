package au10_test

import (
	"reflect"
	"testing"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_Message(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	data := &au10.MessageData{ID: 653, Kind: 345, Size: 987}
	assert.Equal(3, reflect.Indirect(reflect.ValueOf(data)).NumField())

	post := mock_au10.NewMockPost(mock)

	message := au10.NewMessage(data, post)

	assert.Equal(data.ID, message.GetID())
	assert.Equal(data.Kind, message.GetKind())
	assert.Equal(data.Size, message.GetSize())
	assert.EqualError(message.Load(nil, 0), "not implemented")
}
