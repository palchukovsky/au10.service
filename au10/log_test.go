package au10_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	"github.com/Shopify/sarama"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_Log(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	writer := mock_au10.NewMockStreamAsyncWriter(mock)
	writer.EXPECT().Close()

	factory := mock_au10.NewMockFactory(mock)

	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)
	service.EXPECT().GetNodeName().MinTimes(1).Return("test node name")

	factory.EXPECT().NewStreamAsyncWriter("log", service).
		Return(nil, errors.New("writer test error"))
	log, err := au10.NewFactory().NewLog(service)
	assert.Nil(log)
	assert.EqualError(err, "writer test error")

	factory.EXPECT().NewStreamAsyncWriter("log", service).Return(writer, nil)
	log, err = au10.NewFactory().NewLog(service)
	assert.NotNil(log)
	defer log.Close()
	assert.NoError(err)

	assert.Equal(au10.Group{Domain: "", Name: ""}, log.GetMembership().Get())

	writer.EXPECT().PushAsync(au10.LogRecordData{
		Node:     "test node name",
		Severity: 0,
		Text:     `test log debug record`}).Return(nil)
	log.Debug("test log %s record", "debug")

	writer.EXPECT().PushAsync(au10.LogRecordData{
		Node:     "test node name",
		Severity: 1,
		Text:     `test log info record`}).Return(nil)
	log.Info("test log %s record", "info")

	writer.EXPECT().PushAsync(au10.LogRecordData{
		Node:     "test node name",
		Severity: 2,
		Text:     `test log warn record`}).Return(nil)
	log.Warn("test log %s record", "warn")

	writer.EXPECT().PushAsync(au10.LogRecordData{
		Node:     "test node name",
		Severity: 3,
		Text:     "test log error record"}).Return(nil)
	log.Error("test log %s record", "error")

	writer.EXPECT().PushAsync(au10.LogRecordData{
		Node:     "test node name",
		Severity: 4,
		Text:     "test log panic record"}).Return(nil)
	assert.PanicsWithValue("test log panic record", func() {
		log.Fatal("test log %s record", "panic")
	})

	writer.EXPECT().PushAsync(au10.LogRecordData{
		Node:     "test node name",
		Severity: 3,
		Text:     "test log error record 2"}).Return(errors.New("test error"))
	log.Error("test log %s record 2", "error")
}

func Test_Au10_NewLogRecord(test *testing.T) {
	assert := assert.New(test)
	now := time.Now()

	record, err := au10.ConvertSaramaMessageIntoLogRecord(&sarama.ConsumerMessage{
		Key: []byte("test key"),
		Value: []byte(
			`{"n": "test node name", "s": 2, "t": "test log record test"}`),
		Offset:    123,
		Timestamp: now})
	if !assert.NotNil(record) {
		return
	}
	assert.NoError(err)
	assert.Equal(int64(123), record.GetSequenceNumber())
	assert.True(now.Equal(record.GetTime()))
	assert.Equal("test log record test", record.GetText())
	assert.Equal("warn", record.GetSeverity())
	assert.Equal("test key", record.GetNodeType())
	assert.Equal("test node name", record.GetNodeName())

	record, err = au10.ConvertSaramaMessageIntoLogRecord(&sarama.ConsumerMessage{
		Key:       []byte("test key"),
		Value:     []byte(`{`),
		Offset:    123,
		Timestamp: now})
	assert.Nil(record)
	assert.NotNil(err)
}

func Test_Au10_LogReader(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)

	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	reader := mock_au10.NewMockStreamReader(mock)
	reader.EXPECT().Close()
	var convertMessage func(*sarama.ConsumerMessage) (interface{}, error)
	factory.EXPECT().NewStreamReader([]string{"log"}, gomock.Any(), service).
		Do(func(topics []string,
			convertMessageCallback func(*sarama.ConsumerMessage) (interface{}, error),
			service au10.Service) {

			convertMessage = convertMessageCallback
		}).
		Return(reader)

	log := au10.NewLogReader(service)
	assert.NotNil(log)
	defer log.Close()

	assert.Equal(au10.Group{Domain: "", Name: ""}, log.GetMembership().Get())

	reader.EXPECT().NewSubscription(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("test create subscription error"))
	subscription, err := log.Subscribe()
	assert.Nil(subscription)
	assert.EqualError(err, "test create subscription error")

	if !assert.NotNil(convertMessage) {
		return
	}
	now := time.Now()
	var convertedMessage interface{}
	convertedMessage, err = convertMessage(&sarama.ConsumerMessage{
		Key: []byte("test key"),
		Value: []byte(
			`{"n": "test node name", "s": 2, "t": "test log record test"}`),
		Offset:    123,
		Timestamp: now})
	assert.NoError(err)
	assert.Equal(int64(123),
		convertedMessage.(au10.LogRecord).GetSequenceNumber())
	assert.True(now.Equal(convertedMessage.(au10.LogRecord).GetTime()))
	assert.Equal("test log record test",
		convertedMessage.(au10.LogRecord).GetText())
	assert.Equal("warn", convertedMessage.(au10.LogRecord).GetSeverity())
	assert.Equal("test key", convertedMessage.(au10.LogRecord).GetNodeType())
	assert.Equal("test node name",
		convertedMessage.(au10.LogRecord).GetNodeName())

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
	subscription, err = log.Subscribe()
	if !assert.NotNil(subscription) {
		return
	}
	defer subscription.Close()
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
	go func() {
		receivedRecord := false
		for {
			select {
			case record := <-subscription.GetRecordsChan():
				assert.Equal(int64(123), record.GetSequenceNumber())
				assert.False(receivedRecord)
				receivedRecord = true
				messageBarrier.Done()
			case err := <-subscription.GetErrChan():
				assert.NoError(err)
				assert.True(receivedRecord)
				stopBarrier.Done()
				return
			}
		}
	}()
	record, _ := au10.ConvertSaramaMessageIntoLogRecord(&sarama.ConsumerMessage{
		Value:  []byte(`{}`),
		Offset: 123})
	handleSubscription(record)
	messageBarrier.Wait()
	subscriptionErrChan <- nil
	stopBarrier.Wait()
}
