package au10_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	mock_sarama "bitbucket.org/au10/service/mock/sarama"
	"github.com/Shopify/sarama"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_StreamWriter_MessageAsync(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)

	producer := mock_sarama.NewMockAsyncProducer(ctrl)
	producerErrsChan := make(chan *sarama.ProducerError)
	producer.EXPECT().Close().Do(func() { close(producerErrsChan) })
	producer.EXPECT().Errors().Return(producerErrsChan)

	factory := mock_au10.NewMockFactory(ctrl)

	log := mock_au10.NewMockLog(ctrl)

	service := mock_au10.NewMockService(ctrl)
	factory.EXPECT().NewSaramaProducer(service, false).Return(producer, nil)
	service.EXPECT().GetFactory().Return(factory)
	service.EXPECT().GetNodeType().MinTimes(1).Return("test node type")
	service.EXPECT().Log().Return(log)

	stream, err := au10.NewFactory().NewStreamWriter("test topic 2", service)
	assert.NotNil(stream)
	assert.NoError(err)

	messagesChan := make(chan *sarama.ProducerMessage, 1)
	defer close(messagesChan)

	// failed to serialize message:
	wrongMessage := make(chan interface{})
	defer close(wrongMessage)
	err = stream.PushAsync(wrongMessage)
	assert.NotNil(err)

	producer.EXPECT().Input().Return(messagesChan)
	err = stream.PushAsync(struct {
		Field1 string
		Field2 int
	}{Field1: "test field value", Field2: 543})
	assert.NoError(err)

	message := <-messagesChan
	assert.Equal("test topic 2", message.Topic)
	assert.Equal(sarama.StringEncoder("test node type"), message.Key)
	var messageValue struct {
		Field1 string
		Field2 int
	}
	var messageValueEncoded []byte
	messageValueEncoded, err = message.Value.Encode()
	assert.NoError(err)
	err = json.Unmarshal(messageValueEncoded, &messageValue)
	assert.NoError(err)
	assert.Equal("test field value", messageValue.Field1)
	assert.Equal(543, messageValue.Field2)

	stream.Close()
}

func Test_Au10_StreamWriter_MessageSync(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)

	producer := mock_sarama.NewMockAsyncProducer(ctrl)
	producerErrsChan := make(chan *sarama.ProducerError)
	producerSuncessesChan := make(chan *sarama.ProducerMessage)
	producer.EXPECT().Close()
	producer.EXPECT().Errors().Return(producerErrsChan)
	producer.EXPECT().Successes().Return(producerSuncessesChan)

	factory := mock_au10.NewMockFactory(ctrl)

	log := mock_au10.NewMockLog(ctrl)

	service := mock_au10.NewMockService(ctrl)
	factory.EXPECT().NewSaramaProducer(service, true).Return(producer, nil)
	service.EXPECT().GetFactory().Return(factory)
	service.EXPECT().GetNodeType().MinTimes(1).Return("test node type")
	service.EXPECT().Log().Return(log)

	stream, err := au10.NewFactory().NewStreamWriterWithResult("test topic 2",
		service)
	assert.NotNil(stream)
	assert.NoError(err)

	messagesChan := make(chan *sarama.ProducerMessage)
	defer close(messagesChan)

	// failed to serialize message:
	wrongMessage := make(chan interface{})
	var id uint32
	var timestamp time.Time
	id, timestamp, err = stream.Push(wrongMessage)
	close(wrongMessage)
	assert.NotNil(err)
	assert.Equal(uint32(0), id)
	assert.True(time.Time{}.Equal(timestamp))

	//	success
	var message *sarama.ProducerMessage
	now := time.Now()
	go func() {
		message = <-messagesChan
		producerSuncessesChan <- &sarama.ProducerMessage{
			Offset:    876,
			Timestamp: now}
	}()
	producer.EXPECT().Input().Return(messagesChan)
	id, timestamp, err = stream.Push(struct {
		Field1 string
		Field2 int
	}{Field1: "test field value", Field2: 543})
	assert.NoError(err)
	assert.Equal(uint32(876), id)
	assert.Equal(now, timestamp)
	if !assert.NotNil(message) {
		return
	}
	assert.Equal("test topic 2", message.Topic)
	assert.Equal(sarama.StringEncoder("test node type"), message.Key)
	var messageValue struct {
		Field1 string
		Field2 int
	}
	var messageValueEncoded []byte
	messageValueEncoded, err = message.Value.Encode()
	assert.NoError(err)
	err = json.Unmarshal(messageValueEncoded, &messageValue)
	assert.NoError(err)
	assert.Equal("test field value", messageValue.Field1)
	assert.Equal(543, messageValue.Field2)

	// error
	go func() {
		<-messagesChan
		producerErrsChan <- &sarama.ProducerError{Err: errors.New("test error")}
	}()
	producer.EXPECT().Input().Return(messagesChan)
	producer.EXPECT().Errors().Return(producerErrsChan)
	producer.EXPECT().Successes().Return(producerSuncessesChan)
	id, timestamp, err = stream.Push(struct{}{})
	assert.EqualError(err, "test error")
	assert.Equal(uint32(0), id)
	assert.Equal(time.Time{}, timestamp)

	// closed success chan
	go func() {
		<-messagesChan
		close(producerSuncessesChan)
	}()
	producer.EXPECT().Input().Return(messagesChan)
	producer.EXPECT().Errors().Return(producerErrsChan)
	producer.EXPECT().Successes().Return(producerSuncessesChan)
	id, timestamp, err = stream.Push(struct{}{})
	assert.EqualError(err, "stream closed")
	assert.Equal(uint32(0), id)
	assert.Equal(time.Time{}, timestamp)

	// closed errors chan
	producerSuncessesChan = make(chan *sarama.ProducerMessage)
	go func() {
		<-messagesChan
		close(producerErrsChan)
	}()
	producer.EXPECT().Input().Return(messagesChan)
	producer.EXPECT().Errors().Return(producerErrsChan)
	producer.EXPECT().Successes().Return(producerSuncessesChan)
	id, timestamp, err = stream.Push(struct{}{})
	assert.EqualError(err, "stream closed")
	assert.Equal(uint32(0), id)
	assert.Equal(time.Time{}, timestamp)

	close(producerSuncessesChan)
	stream.Close()
}

func Test_Au10_StreamWriter_FailedToNew(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(ctrl)
	log := mock_au10.NewMockLog(ctrl)
	service := mock_au10.NewMockService(ctrl)

	factory.EXPECT().NewSaramaProducer(service, false).
		Return(nil, errors.New("test error 1"))
	service.EXPECT().Log().Return(log)
	service.EXPECT().GetFactory().Return(factory)
	service.EXPECT().GetNodeType().Return("test node type")
	streamAsync, err := au10.NewFactory().NewStreamWriter("test topic 1", service)
	assert.Nil(streamAsync)
	assert.EqualError(err, "test error 1")

	factory.EXPECT().NewSaramaProducer(service, true).
		Return(nil, errors.New("test error 2"))
	service.EXPECT().Log().Return(log)
	service.EXPECT().GetFactory().Return(factory)
	service.EXPECT().GetNodeType().Return("test node type")
	var streamSync au10.StreamWriterWithResult
	streamSync, err = au10.NewFactory().NewStreamWriterWithResult("test topic 2",
		service)
	assert.Nil(streamSync)
	assert.EqualError(err, "test error 2")
}

func Test_Au10_StreamWriter_AsyncErrorsInProducer(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)

	producer := mock_sarama.NewMockAsyncProducer(ctrl)

	factory := mock_au10.NewMockFactory(ctrl)

	log := mock_au10.NewMockLog(ctrl)

	service := mock_au10.NewMockService(ctrl)
	service.EXPECT().GetNodeType().MinTimes(2).Return("test node type")
	service.EXPECT().Log().MinTimes(1).Return(log)

	// log stream has to use standard go log at error to avoid sequence of calls
	// without end:
	producerErrsChan := make(chan *sarama.ProducerError)
	producer.EXPECT().Close().Do(func() { close(producerErrsChan) }).
		Return(errors.New("test error at closing for log"))
	producer.EXPECT().Errors().Return(producerErrsChan)
	factory.EXPECT().NewSaramaProducer(service, false).Return(producer, nil)
	service.EXPECT().GetFactory().Return(factory)
	stream, err := au10.NewFactory().NewStreamWriter("log", service)
	assert.NotNil(stream)
	assert.NoError(err)
	producerErrsChan <- &sarama.ProducerError{
		Msg: nil,
		Err: errors.New("test producer error for log topic")}
	stream.Close()

	// usual log:
	factory.EXPECT().NewSaramaProducer(service, false).Return(producer, nil)
	service.EXPECT().GetFactory().Return(factory)
	producerErrsChan = make(chan *sarama.ProducerError)
	producer.EXPECT().Close().Do(func() { close(producerErrsChan) }).
		Return(errors.New("test error at closing for non-log"))
	producer.EXPECT().Errors().Return(producerErrsChan)
	stream, err = au10.NewFactory().NewStreamWriter("not log", service)
	assert.NotNil(stream)
	assert.NoError(err)
	log.EXPECT().Error(
		`Stream "not log" writing error: "test producer error for non-log topic".`)
	producerErrsChan <- &sarama.ProducerError{
		Msg: nil,
		Err: errors.New("test producer error for non-log topic")}
	log.EXPECT().Error(
		`Failed to close stream writing "not log": "test error at closing for non-log".`)
	stream.Close()
}
