package au10_test

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	"github.com/Shopify/sarama"
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

func Test_Au10_Publisher_AddVocal(test *testing.T) {
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
	assert.NoError(err)
	assert.NotNil(publisher)

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

	writer.EXPECT().Push(vocalDeclaration).Return(uint32(123), now, nil)
	vocal, err := publisher.AddVocal(vocalDeclaration)
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
		Return(uint32(123), now, errors.New("test error"))
	vocal, err = publisher.AddVocal(vocalDeclaration)
	assert.Nil(vocal)
	assert.EqualError(err, "test error")

	publisher.Close()
}

func Test_Au10_Publisher_PublishStreamSingleton(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)

	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	reader := mock_au10.NewMockStreamReader(mock)
	var convertMessage func(*sarama.ConsumerMessage) (interface{}, error)
	factory.EXPECT().NewStreamReader([]string{"pubs"}, gomock.Any(), service).
		Times(2).
		Do(func(topics []string,
			convertMessageCallback func(*sarama.ConsumerMessage) (interface{}, error),
			service au10.Service) {
			convertMessage = convertMessageCallback
		}).
		Return(reader)

	reader.EXPECT().NewSubscription(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("test create subscription error"))
	reader.EXPECT().Close()
	stream, err := au10.NewPublishStreamSingleton(service)
	assert.Nil(stream)
	assert.EqualError(err, "test create subscription error")

	if !assert.NotNil(convertMessage) {
		return
	}
	now := time.Now()
	var convertedMessage interface{}
	convertedMessage, err = convertMessage(&sarama.ConsumerMessage{
		Key:       []byte("test key"),
		Value:     []byte(`{`),
		Offset:    123,
		Timestamp: now})
	assert.Nil(convertedMessage)
	assert.EqualError(err,
		`failed to parse vocal declaration: "unexpected end of JSON input"`)
	convertedMessage, err = convertMessage(&sarama.ConsumerMessage{
		Key: []byte("test key"),
		Value: []byte(`{
			"a": 8765,
			"l": {"x": 234.567, "y": 987.654},
			"m": [{"k": 0, "s": 678}, {"k": 0, "s": 987}]
		}`),
		Offset:    123,
		Timestamp: now})
	assert.Equal(1,
		reflect.Indirect(reflect.ValueOf(&au10.VocalDeclaration{})).NumField())
	assert.Equal(3,
		reflect.Indirect(reflect.ValueOf(&au10.PostDeclaration{})).NumField())
	assert.Equal(2,
		reflect.Indirect(reflect.ValueOf(&au10.GeoPoint{})).NumField())
	assert.Equal(2,
		reflect.Indirect(reflect.ValueOf(&au10.MessageDeclaration{})).NumField())
	assert.NoError(err)
	convertedVocal := convertedMessage.(au10.Vocal)
	assert.Equal(au10.PostID(123), convertedVocal.GetID())
	assert.True(now.Equal(convertedVocal.GetTime()))
	assert.Equal(au10.GeoPoint{Latitude: 234.567, Longitude: 987.654},
		*convertedVocal.GetLocation())
	assert.Equal(au10.UserID(8765), convertedVocal.GetAuthor())
	messages := convertedVocal.GetMessages()
	if assert.Equal(2, len(messages)) {
		assert.Equal(au10.MessageID(0), messages[0].GetID())
		assert.Equal(au10.MessageKindText, messages[0].GetKind())
		assert.Equal(uint32(678), messages[0].GetSize())
		assert.Equal(au10.MessageID(1), messages[1].GetID())
		assert.Equal(au10.MessageKindText, messages[1].GetKind())
		assert.Equal(uint32(987), messages[1].GetSize())
	}

	streamSubscription := mock_au10.NewMockStreamSubscription(mock)
	streamSubscription.EXPECT().Close()
	var handleSubscription func(interface{})
	var subscriptionErrChan chan<- error
	reader.EXPECT().NewSubscription(gomock.Any(), gomock.Any()).
		Do(func(handle func(interface{}), errChan chan<- error) {
			handleSubscription = handle
			subscriptionErrChan = errChan
		}).
		Return(streamSubscription, nil)
	reader.EXPECT().Close()
	stream, err = au10.NewPublishStreamSingleton(service)
	if !assert.NotNil(stream) {
		return
	}
	defer stream.Close()
	assert.NoError(err)
	if !assert.NotNil(handleSubscription) {
		return
	}
	if !assert.NotNil(subscriptionErrChan) {
		return
	}

	messageBarrier := sync.WaitGroup{}
	messageBarrier.Add(1)
	stopBarrier := sync.WaitGroup{}
	stopBarrier.Add(1)
	vocal := mock_au10.NewMockVocal(mock)
	go func() {
		receivedRecord := false
		for {
			select {
			case record := <-stream.GetRecordsChan():
				assert.True(vocal == record)
				assert.False(receivedRecord)
				receivedRecord = true
				messageBarrier.Done()
			case err := <-stream.GetErrChan():
				assert.NoError(err)
				assert.True(receivedRecord)
				stopBarrier.Done()
				return
			}
		}
	}()
	handleSubscription(vocal)
	messageBarrier.Wait()
	subscriptionErrChan <- nil
	stopBarrier.Wait()
}
