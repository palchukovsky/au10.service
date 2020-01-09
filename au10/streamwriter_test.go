package au10_test

import (
	"encoding/json"
	"errors"
	"testing"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	mock_sarama "bitbucket.org/au10/service/mock/sarama"
	"github.com/Shopify/sarama"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_StreamWriter_Message(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)

	producer := mock_sarama.NewMockAsyncProducer(ctrl)
	producerErrsChan := make(chan *sarama.ProducerError)
	producer.EXPECT().Close().Do(func() { close(producerErrsChan) })
	producer.EXPECT().Errors().Return(producerErrsChan)

	factory := mock_au10.NewMockFactory(ctrl)

	service := mock_au10.NewMockService(ctrl)
	factory.EXPECT().NewSaramaProducer(service).Return(producer, nil)
	service.EXPECT().GetFactory().Return(factory)
	service.EXPECT().GetNodeType().MinTimes(1).Return("test node type")

	stream, err := au10.NewFactory().NewStreamWriter("test topic 2", service)
	assert.NotNil(stream)
	assert.NoError(err)

	messagesChan := make(chan *sarama.ProducerMessage, 1)
	defer close(messagesChan)

	producer.EXPECT().Input().Return(messagesChan)
	err = stream.Push(struct {
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

	// failed to serialize message:
	wrongMessage := make(chan interface{})
	defer close(wrongMessage)
	err = stream.Push(wrongMessage)
	assert.NotNil(err)

	stream.Close()
}

func Test_Au10_StreamWriter_FailedToNew(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(ctrl)

	service := mock_au10.NewMockService(ctrl)
	factory.EXPECT().NewSaramaProducer(service).
		Return(nil, errors.New("test error"))
	service.EXPECT().GetFactory().Return(factory)
	service.EXPECT().GetNodeType().Return("test node type")

	stream, err := au10.NewFactory().
		NewStreamWriter("test topic 1", service)
	assert.Nil(stream)
	assert.EqualError(err, "test error")
}

func Test_Au10_StreamWriter_ErrorsInProducer(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)

	producer := mock_sarama.NewMockAsyncProducer(ctrl)

	factory := mock_au10.NewMockFactory(ctrl)

	service := mock_au10.NewMockService(ctrl)
	service.EXPECT().GetNodeType().MinTimes(2).Return("test node type")

	// log stream has to use standard go log at error to avoid sequence of calls
	// without end:
	producerErrsChan := make(chan *sarama.ProducerError)
	producer.EXPECT().Close().Do(func() { close(producerErrsChan) }).
		Return(errors.New("test error at closing for log"))
	producer.EXPECT().Errors().Return(producerErrsChan)
	factory.EXPECT().NewSaramaProducer(service).Return(producer, nil)
	service.EXPECT().GetFactory().Return(factory)
	stream, err := au10.NewFactory().NewStreamWriter("log", service)
	assert.NotNil(stream)
	assert.NoError(err)
	producerErrsChan <- &sarama.ProducerError{
		Msg: nil,
		Err: errors.New("test producer error for log topic")}
	stream.Close()

	// usual log:
	factory.EXPECT().NewSaramaProducer(service).Return(producer, nil)
	service.EXPECT().GetFactory().Return(factory)
	producerErrsChan = make(chan *sarama.ProducerError)
	producer.EXPECT().Close().Do(func() { close(producerErrsChan) }).
		Return(errors.New("test error at closing for non-log"))
	producer.EXPECT().Errors().Return(producerErrsChan)
	stream, err = au10.NewFactory().NewStreamWriter("not log", service)
	assert.NotNil(stream)
	assert.NoError(err)
	log := mock_au10.NewMockLog(ctrl)
	log.EXPECT().Error(
		`Stream "not log" writing error: "test producer error for non-log topic".`)
	service.EXPECT().Log().Return(log)
	producerErrsChan <- &sarama.ProducerError{
		Msg: nil,
		Err: errors.New("test producer error for non-log topic")}
	service.EXPECT().Log().Return(log)
	log.EXPECT().Error(
		`Failed to close stream writing "not log": "test error at closing for non-log".`)
	stream.Close()
}
