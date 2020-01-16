package au10_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_Publisher_Fields(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)
	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().Return(factory)

	log := mock_au10.NewMockLog(mock)
	service.EXPECT().Log().Return(log)

	factory.EXPECT().NewStreamWriterWithResult("pubs", service).
		Return(nil, errors.New("writer test error"))
	publisher, err := au10.NewPublisher(service)
	assert.Nil(publisher)
	assert.EqualError(err, "writer test error")

	writer := mock_au10.NewMockStreamWriterWithResult(mock)
	writer.EXPECT().Close()
	factory.EXPECT().NewStreamWriterWithResult("pubs", service).
		Return(writer, nil)
	service.EXPECT().GetFactory().Return(factory)
	publisher, err = au10.NewPublisher(service)
	assert.NotNil(publisher)
	defer publisher.Close()
	assert.NoError(err)

	assert.Equal(au10.NewMembership("", ""), publisher.GetMembership())
}

func Test_Au10_Publisher_PublishVocal(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)
	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().Return(factory)

	log := mock_au10.NewMockLog(mock)
	service.EXPECT().Log().Return(log)

	writer := mock_au10.NewMockStreamWriterWithResult(mock)
	writer.EXPECT().Close()
	factory.EXPECT().NewStreamWriterWithResult("pubs", service).
		Return(writer, nil)
	publisher, err := au10.NewPublisher(service)
	now := time.Now()

	location := &au10.GeoPoint{}
	vocalDeclaration := &au10.VocalDeclaration{
		au10.PostDeclaration{
			Author:   987123,
			Location: location,
			Messages: []*au10.MessageDeclaration{
				&au10.MessageDeclaration{Kind: au10.MessageKindText, Size: 567},
				&au10.MessageDeclaration{Kind: au10.MessageKindText, Size: 987}}}}
	assert.Equal(1,
		reflect.Indirect(reflect.ValueOf(vocalDeclaration)).NumField())
	assert.Equal(3,
		reflect.Indirect(reflect.ValueOf(&au10.PostDeclaration{})).NumField())
	assert.Equal(2,
		reflect.Indirect(reflect.ValueOf(&au10.MessageDeclaration{})).NumField())

	writer.EXPECT().Push(vocalDeclaration).Return(uint64(123), now, nil)
	vocal, err := publisher.PublishVocal(vocalDeclaration)
	assert.NotNil(vocal)
	assert.NoError(err)
	assert.Equal(au10.PostID(123), vocal.GetID())
	assert.Equal(now.UnixNano(), vocal.GetTime().UnixNano())
	assert.Equal(au10.UserID(987123), vocal.GetAuthor())
	assert.True(location == vocal.GetLocation())
	if assert.Equal(len(vocalDeclaration.Messages), len(vocal.GetMessages())) {
		for i, m := range vocal.GetMessages() {
			assert.Equal(au10.MessageID(i), m.GetID())
			assert.Equal(vocalDeclaration.Messages[i].Kind, m.GetKind())
			assert.Equal(vocalDeclaration.Messages[i].Size, m.GetSize())
		}
	}

	writer.EXPECT().Push(vocalDeclaration).
		Return(uint64(123), now, errors.New("test error"))
	vocal, err = publisher.PublishVocal(vocalDeclaration)
	assert.Nil(vocal)
	assert.EqualError(err, "test error")

	publisher.Close()
}
