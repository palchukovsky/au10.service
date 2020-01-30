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
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	service.EXPECT().Log().MinTimes(1).Return(mock_au10.NewMockLog(mock))

	syncWriter := mock_au10.NewMockStreamSyncWriter(mock)
	asyncWriter := mock_au10.NewMockStreamAsyncWriter(mock)

	factory.EXPECT().NewStreamSyncWriter("pub-posts", service).
		Return(syncWriter, errors.New("writer test error"))
	publisher, err := au10.NewPublisher(service)
	assert.Nil(publisher)
	assert.EqualError(err, "writer test error")

	syncWriter.EXPECT().Close()
	factory.EXPECT().NewStreamSyncWriter("pub-posts", service).
		Return(syncWriter, nil)
	factory.EXPECT().NewStreamAsyncWriter("pub-messages", service).
		Return(asyncWriter, errors.New("writer test error 2"))
	publisher, err = au10.NewPublisher(service)
	assert.Nil(publisher)
	assert.EqualError(err, "writer test error 2")

	syncWriter.EXPECT().Close()
	factory.EXPECT().NewStreamSyncWriter("pub-posts", service).
		Return(syncWriter, nil)
	asyncWriter.EXPECT().Close()
	factory.EXPECT().NewStreamAsyncWriter("pub-messages", service).
		Return(asyncWriter, nil)
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
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	service.EXPECT().Log().Return(mock_au10.NewMockLog(mock))

	syncWriter := mock_au10.NewMockStreamSyncWriter(mock)
	syncWriter.EXPECT().Close()
	factory.EXPECT().NewStreamSyncWriter("pub-posts", service).
		Return(syncWriter, nil)
	asyncWriter := mock_au10.NewMockStreamAsyncWriter(mock)
	asyncWriter.EXPECT().Close()
	factory.EXPECT().NewStreamAsyncWriter("pub-messages", service).
		Return(asyncWriter, nil)

	publisher, err := au10.NewPublisher(service)
	assert.NoError(err)
	assert.NotNil(publisher)
	defer publisher.Close()

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

	syncWriter.EXPECT().Push(vocalDeclaration).Return(uint32(123), now, nil)
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

	syncWriter.EXPECT().Push(vocalDeclaration).
		Return(uint32(123), now, errors.New("test error"))
	vocal, err = publisher.AddVocal(vocalDeclaration)
	assert.Nil(vocal)
	assert.EqualError(err, "test error")
}

func Test_Au10_Publisher_AppendMessage(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)
	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	service.EXPECT().Log().Return(mock_au10.NewMockLog(mock))

	syncWriter := mock_au10.NewMockStreamSyncWriter(mock)
	syncWriter.EXPECT().Close()
	factory.EXPECT().NewStreamSyncWriter("pub-posts", service).
		Return(syncWriter, nil)
	asyncWriter := mock_au10.NewMockStreamAsyncWriter(mock)
	asyncWriter.EXPECT().Close()
	factory.EXPECT().NewStreamAsyncWriter("pub-messages", service).
		Return(asyncWriter, nil)

	user := mock_au10.NewMockUser(mock)
	user.EXPECT().GetID().Return(au10.UserID(999111))

	publisher, _ := au10.NewPublisher(service)
	defer publisher.Close()

	data := []byte("test data")

	asyncWriter.EXPECT().
		PushAsync(&au10.PublisherMessageData{
			Post:   au10.PostID(123),
			ID:     au10.MessageID(456),
			Author: au10.UserID(999111),
			Data:   data}).
		Return(errors.New("test error"))
	err := publisher.AppendMessage(
		au10.MessageID(456), au10.PostID(123), user, data)
	assert.EqualError(err, "test error")
}

func testPublishStreamSingleton(
	test *testing.T,
	topicName, entityName string,
	newStream func(au10.Service) (interface{}, error),
	closeStream func(interface{}),
	jsonDoc string,
	testConvertedMessage func(
		assert *assert.Assertions,
		time time.Time,
		convertedMessage interface{}),
	newEntity func(*gomock.Controller) interface{},
	runRead func(
		*assert.Assertions,
		interface{},
		*sync.WaitGroup,
		*sync.WaitGroup,
		interface{})) {

	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)

	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	reader := mock_au10.NewMockStreamReader(mock)
	var convertMessage func(*sarama.ConsumerMessage) (interface{}, error)
	factory.EXPECT().NewStreamReader([]string{topicName}, gomock.Any(), service).
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
	stream, err := newStream(service)
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
		"failed to parse "+entityName+`: "unexpected end of JSON input"`)
	convertedMessage, err = convertMessage(&sarama.ConsumerMessage{
		Key:       []byte("test key"),
		Value:     []byte(jsonDoc),
		Offset:    123,
		Timestamp: now})
	assert.NoError(err)
	testConvertedMessage(assert, now, convertedMessage)

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
	stream, err = newStream(service)
	if !assert.NotNil(stream) {
		return
	}
	defer closeStream(stream)
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
	entity := newEntity(mock)
	go runRead(assert, stream, &stopBarrier, &messageBarrier, entity)
	handleSubscription(entity)
	messageBarrier.Wait()
	subscriptionErrChan <- nil
	stopBarrier.Wait()
}
func Test_Au10_PublishPostsStreamSingleton(test *testing.T) {
	testPublishStreamSingleton(
		test,
		"pub-posts",
		"vocal declaration",
		func(service au10.Service) (interface{}, error) {
			return au10.NewPublishPostsStreamSingleton(service)
		},
		func(stream interface{}) { stream.(au10.PublishPostsStream).Close() },
		`{
			"a": 8765,
			"l": {"x": 234.567, "y": 987.654},
			"m": [{"k": 0, "s": 678}, {"k": 0, "s": 987}]
		}`,
		func(
			assert *assert.Assertions,
			time time.Time,
			convertedMessage interface{}) {
			assert.Equal(1,
				reflect.Indirect(reflect.ValueOf(&au10.VocalDeclaration{})).NumField())
			assert.Equal(3,
				reflect.Indirect(reflect.ValueOf(&au10.PostDeclaration{})).NumField())
			assert.Equal(2,
				reflect.Indirect(reflect.ValueOf(&au10.GeoPoint{})).NumField())
			assert.Equal(2,
				reflect.Indirect(reflect.ValueOf(&au10.MessageDeclaration{})).NumField())
			convertedVocal := convertedMessage.(au10.Vocal)
			assert.Equal(au10.PostID(123), convertedVocal.GetID())
			assert.True(time.Equal(convertedVocal.GetTime()))
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
		},
		func(mock *gomock.Controller) interface{} {
			return mock_au10.NewMockVocal(mock)
		},
		func(
			assert *assert.Assertions,
			stream interface{},
			stop *sync.WaitGroup,
			message *sync.WaitGroup,
			entity interface{}) {
			receivedRecord := false
			for {
				select {
				case record := <-stream.(au10.PublishPostsStream).GetRecordsChan():
					assert.True(entity == record)
					assert.False(receivedRecord)
					receivedRecord = true
					message.Done()
				case err := <-stream.(au10.PublishPostsStream).GetErrChan():
					assert.NoError(err)
					assert.True(receivedRecord)
					stop.Done()
					return
				}
			}
		})
}

func Test_Au10_PublishMessagesStreamSingleton(test *testing.T) {
	testPublishStreamSingleton(
		test,
		"pub-messages",
		"message data",
		func(service au10.Service) (interface{}, error) {
			return au10.NewPublishMessagesStreamSingleton(service)
		},
		func(stream interface{}) { stream.(au10.PublishMessagesStream).Close() },
		`{
			"p": 123,
			"m": 678,
			"a": 999111,
			"d": "YXBwbGU="
		}`,
		func(
			assert *assert.Assertions,
			time time.Time,
			convertedMessage interface{}) {
			assert.Equal(4,
				reflect.Indirect(reflect.ValueOf(&au10.PublisherMessageData{})).NumField())
			data := convertedMessage.(*au10.PublisherMessageData)
			assert.Equal(au10.PostID(123), data.Post)
			assert.Equal(au10.MessageID(678), data.ID)
			assert.Equal(au10.UserID(999111), data.Author)
			assert.Equal([]byte("apple"), data.Data)
		},
		func(*gomock.Controller) interface{} {
			return &au10.PublisherMessageData{}
		},
		func(
			assert *assert.Assertions,
			stream interface{},
			stop *sync.WaitGroup,
			message *sync.WaitGroup,
			entity interface{}) {
			receivedRecord := false
			for {
				select {
				case record := <-stream.(au10.PublishMessagesStream).GetRecordsChan():
					assert.True(entity == record)
					assert.False(receivedRecord)
					receivedRecord = true
					message.Done()
				case err := <-stream.(au10.PublishMessagesStream).GetErrChan():
					assert.NoError(err)
					assert.True(receivedRecord)
					stop.Done()
					return
				}
			}
		})
}
