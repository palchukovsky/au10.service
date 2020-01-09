package au10_test

import (
	"errors"
	"testing"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_Publisher(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)
	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().Return(factory)

	factory.EXPECT().NewStreamWriter("pubs", service).
		Return(nil, errors.New("writer test error"))
	publisher, err := au10.NewPublisher(service)
	assert.Nil(publisher)
	assert.EqualError(err, "writer test error")

	writer := mock_au10.NewMockStreamWriter(mock)
	writer.EXPECT().Close()
	factory.EXPECT().NewStreamWriter("pubs", service).Return(writer, nil)
	service.EXPECT().GetFactory().Return(factory)
	publisher, err = au10.NewPublisher(service)
	assert.NotNil(publisher)
	defer publisher.Close()
	assert.NoError(err)

	assert.Equal(au10.NewMembership("", ""), publisher.GetMembership())

	vocal, err := publisher.PublishVocal(nil, nil)
	assert.Nil(vocal)
	assert.EqualError(err, "not implemented")
}
