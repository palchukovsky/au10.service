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

func Test_Au10_Posts_General(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)
	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	reader := mock_au10.NewMockStreamReader(mock)
	factory.EXPECT().NewStreamReader([]string{"posts"}, gomock.Any(), service).
		Do(
			func([]string,
				func(*sarama.ConsumerMessage) (interface{}, error),
				au10.Service) {
			}).
		Return(reader)

	posts := au10.NewPosts(service)
	defer posts.Close()
	reader.EXPECT().Close()

	assert.Equal(au10.NewMembership("", ""), posts.GetMembership())

	post, err := posts.GetPost(au10.PostID(1))
	assert.Nil(post)
	assert.EqualError(err, "post with ID 1 is nonexistent")
}

func Test_Au10_Posts_Subscription(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)
	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	reader := mock_au10.NewMockStreamReader(mock)
	var convertMessage func(*sarama.ConsumerMessage) (interface{}, error)
	factory.EXPECT().NewStreamReader([]string{"posts"}, gomock.Any(), service).
		Do(func(topics []string,
			convertMessageCallback func(*sarama.ConsumerMessage) (interface{}, error),
			service au10.Service) {
			convertMessage = convertMessageCallback
		}).
		Return(reader)

	posts := au10.NewPosts(service)
	defer posts.Close()
	reader.EXPECT().Close()

	reader.EXPECT().NewSubscription(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("test create subscription error"))
	subs, err := posts.Subscribe()
	assert.Nil(subs)
	assert.EqualError(err, "test create subscription error")

	if !assert.NotNil(convertMessage) {
		return
	}
	convertedMessage, err := convertMessage(&sarama.ConsumerMessage{
		Key:       []byte("test key"),
		Value:     []byte(`{`),
		Offset:    123,
		Timestamp: time.Now()})
	assert.Nil(convertedMessage)
	assert.EqualError(err,
		`failed to parse vocal-record: "unexpected end of JSON input"`)
	convertedMessage, err = convertMessage(&sarama.ConsumerMessage{
		Key: []byte("test key"),
		Value: []byte(`{
			"i": 321,
			"t": 8642,
			"a": 8765,
			"l": {"x": 234.567, "y": 987.654},
			"m": [{"i": 10, "k": 0, "s": 678}, {"i": 99, "k": 0, "s": 987}]
		}`),
		Offset:    123,
		Timestamp: time.Now()})
	assert.Equal(5,
		reflect.Indirect(reflect.ValueOf(&au10.PostData{})).NumField())
	assert.Equal(2,
		reflect.Indirect(reflect.ValueOf(&au10.GeoPoint{})).NumField())
	assert.Equal(3,
		reflect.Indirect(reflect.ValueOf(&au10.MessageData{})).NumField())
	assert.NoError(err)
	convertedVocal := convertedMessage.(au10.Vocal)
	assert.Equal(au10.PostID(321), convertedVocal.GetID())
	assert.True(time.Unix(0, 8642).Equal(convertedVocal.GetTime()))
	assert.Equal(au10.GeoPoint{Latitude: 234.567, Longitude: 987.654},
		*convertedVocal.GetLocation())
	assert.Equal(au10.UserID(8765), convertedVocal.GetAuthor())
	messages := convertedVocal.GetMessages()
	if assert.Equal(2, len(messages)) {
		assert.Equal(au10.MessageID(10), messages[0].GetID())
		assert.Equal(au10.MessageKindText, messages[0].GetKind())
		assert.Equal(uint32(678), messages[0].GetSize())
		assert.Equal(au10.MessageID(99), messages[1].GetID())
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
	subs, err = posts.Subscribe()
	if !assert.NotNil(subs) {
		return
	}
	defer subs.Close()
	assert.NoError(err)
	if !assert.NotNil(handleSubscription) {
		return
	}
	if !assert.NotNil(subscriptionErrChan) {
		return
	}

	vocal := mock_au10.NewMockVocal(mock)
	messageBarrier := sync.WaitGroup{}
	messageBarrier.Add(1)
	stopBarrier := sync.WaitGroup{}
	stopBarrier.Add(1)
	go func() {
		receivedRecord := false
		for {
			select {
			case record := <-subs.GetRecordsChan():
				assert.True(vocal == record)
				assert.False(receivedRecord)
				receivedRecord = true
				messageBarrier.Done()
			case err := <-subs.GetErrChan():
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

func Test_Au10_Posts_PostNotifier(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)
	service := mock_au10.NewMockService(mock)
	stream := mock_au10.NewMockStreamWriter(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	factory.EXPECT().NewStreamWriter("posts", service).
		Return(stream, errors.New("test stream error"))
	notifier, err := au10.NewPostNotifier(service)
	assert.Nil(notifier)
	assert.EqualError(err, "test stream error")

	factory.EXPECT().NewStreamWriter("posts", service).Return(stream, nil)
	stream.EXPECT().Close()
	notifier, err = au10.NewPostNotifier(service)
	if !assert.NotNil(notifier) {
		return
	}
	defer notifier.Close()
	assert.NoError(err)

	data := &au10.PostData{ID: 87620}
	post := mock_au10.NewMockVocal(mock)
	post.EXPECT().Export().Return(data)
	stream.EXPECT().PushAsync(data).Return(errors.New("test push error"))
	err = notifier.PushVocal(post)
	assert.EqualError(err, "test push error")
}
