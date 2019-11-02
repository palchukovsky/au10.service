package au10_test

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	mock_sarama "bitbucket.org/au10/service/mock/sarama"
	"github.com/Shopify/sarama"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_StreamReader_Dummy(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)
	service := mock_au10.NewMockService(ctrl)
	stream := au10.CreateFactory().CreateStreamReader(
		[]string{"topic 1", "topic 2"}, nil, service)
	assert.NotNil(stream)
	stream.Close()
}

type streamReaderTest struct {
	test     *testing.T
	topic    []string
	mock     *gomock.Controller
	assert   *assert.Assertions
	factory  *mock_au10.MockFactory
	log      *mock_au10.MockLog
	service  *mock_au10.MockService
	session  *mock_sarama.MockConsumerGroupSession
	consumer *mock_sarama.MockConsumerGroup

	stream au10.StreamReader

	consumeChan         chan sarama.ConsumerGroupClaim
	messagesChanBarrier sync.WaitGroup

	numberOfSubscribers int
}

func createStreamReaderTest(
	test *testing.T,
	convertMessage func(*sarama.ConsumerMessage) (interface{}, error),
	consumerCloseResult error) *streamReaderTest {

	result := &streamReaderTest{
		test:        test,
		topic:       []string{"topic 1", "topic 2"},
		mock:        gomock.NewController(test),
		assert:      assert.New(test),
		consumeChan: make(chan sarama.ConsumerGroupClaim)}

	result.factory = mock_au10.NewMockFactory(result.mock)

	result.log = mock_au10.NewMockLog(result.mock)

	result.service = mock_au10.NewMockService(result.mock)
	result.service.EXPECT().GetFactory().MinTimes(1).Return(result.factory)
	result.service.EXPECT().Log().AnyTimes().Return(result.log)

	result.session = mock_sarama.NewMockConsumerGroupSession(result.mock)

	result.consumer = mock_sarama.NewMockConsumerGroup(result.mock)
	result.consumer.EXPECT().Close().Return(consumerCloseResult)
	if consumerCloseResult == nil {
		result.log.EXPECT().Debug(`Stream reading "%s" closed.`, result.getTopic())
	} else {
		result.log.EXPECT().Error(`Failed to close stream reading "%s": "%s".`,
			result.getTopic(), consumerCloseResult)
	}
	result.consumer.EXPECT().Consume(
		gomock.Any(), result.topic, gomock.Any()).
		AnyTimes().
		DoAndReturn(func(
			ctx context.Context,
			topics []string,
			handler sarama.ConsumerGroupHandler) error {

			// this implementation has empty logic for these methods:
			result.assert.Nil(handler.Setup(nil))
			result.assert.Nil(handler.Cleanup(nil))

			select {
			case message := <-result.consumeChan:
				return handler.ConsumeClaim(result.session, message)
			case <-ctx.Done():
				return nil
			}
		})

	result.factory.EXPECT().CreateSaramaConsumer(result.service).
		Return(result.consumer, nil)

	result.stream = au10.CreateFactory().CreateStreamReader(
		result.topic, convertMessage, result.service)
	result.assert.NotNil(result.stream)

	return result
}

func (test *streamReaderTest) close() {
	test.stream.Close()
	close(test.consumeChan)
	test.mock.Finish()
}

func (test *streamReaderTest) createSubscription(errChan chan<- error) (
	au10.StreamSubscription, *[]uint32) {

	messages := &[]uint32{}
	subscription, err := test.stream.CreateSubscription(
		func(message interface{}) {
			*messages = append(*messages, message.(uint32))
			test.messagesChanBarrier.Done()
		},
		errChan)
	if test.assert.NotNil(subscription) == false {
		return nil, nil
	}
	if test.assert.Nil(err) == false {
		return nil, nil
	}
	test.numberOfSubscribers++
	return subscription, messages
}

func (test *streamReaderTest) closeSubscription(
	subscription au10.StreamSubscription) bool {

	if !test.assert.Less(0, test.numberOfSubscribers) {
		return false
	}
	subscription.Close()
	test.numberOfSubscribers--
	return true
}

func (test *streamReaderTest) send(val uint32, isError bool) {
	message := &sarama.ConsumerMessage{Value: make([]byte, 4)}
	binary.BigEndian.PutUint32(message.Value, val)

	messagesChan := make(chan *sarama.ConsumerMessage, 1)
	defer close(messagesChan)
	if !isError {
		test.messagesChanBarrier.Add(test.numberOfSubscribers)
	}
	messagesChan <- message

	consumerMessage := mock_sarama.NewMockConsumerGroupClaim(test.mock)
	consumerMessage.EXPECT().Messages().Return(messagesChan)
	if !isError {
		test.session.EXPECT().MarkMessage(message, "")
	}
	test.consumeChan <- consumerMessage

	if !isError {
		test.messagesChanBarrier.Wait()
	}
}

func (test *streamReaderTest) getTopic() string {
	return strings.Join(test.topic, ", ")
}

func Test_Au10_StreamReader_1Subscription(t *testing.T) {
	test := createStreamReaderTest(
		t,
		func(message *sarama.ConsumerMessage) (interface{}, error) {
			return binary.BigEndian.Uint32(message.Value), nil
		},
		nil)
	defer test.close()

	errChan := make(chan error)
	defer close(errChan)
	go func() {
		err, isOpened := <-errChan
		test.assert.Nil(err)
		test.assert.False(isOpened)
	}()

	test.log.EXPECT().Debug(`Stream reading "%s" opened.`, test.getTopic())
	subscription, messages := test.createSubscription(errChan)
	if subscription == nil {
		return
	}

	numberOfMessages := 99
	for i := 0; i < numberOfMessages; i++ {
		test.send(uint32(i), false)
	}

	if !test.closeSubscription(subscription) {
		return
	}

	if !test.assert.Equal(numberOfMessages, len(*messages)) {
		return
	}
	for i := 0; i < numberOfMessages; i++ {
		test.assert.Equal(uint32(i), (*messages)[i])
	}

}

func Test_Au10_StreamReader_SeveralSubscriptions(t *testing.T) {
	test := createStreamReaderTest(
		t,
		func(message *sarama.ConsumerMessage) (interface{}, error) {
			return binary.BigEndian.Uint32(message.Value), nil
		},
		nil)
	defer test.close()

	numberOfMessages := 99

	test.log.EXPECT().Debug(`Stream reading "%s" opened.`, test.getTopic())

	errChan := make(chan error)
	defer close(errChan)
	go func() {
		err, isOpened := <-errChan
		test.assert.Nil(err)
		test.assert.False(isOpened)
	}()

	subscription1, messages1 := test.createSubscription(errChan)
	if subscription1 == nil {
		return
	}
	subscription2, messages2 := test.createSubscription(errChan)
	if subscription2 == nil {
		return
	}
	var lastMessage uint32
	for i := 0; i < numberOfMessages; i++ {
		test.send(lastMessage, false)
		lastMessage++
	}

	if !test.closeSubscription(subscription2) {
		return
	}
	for i := 0; i < numberOfMessages; i++ {
		test.send(lastMessage, false)
		lastMessage++
	}

	subscription3, messages3 := test.createSubscription(errChan)
	if subscription3 == nil {
		return
	}
	for i := 0; i < numberOfMessages; i++ {
		test.send(lastMessage, false)
		lastMessage++
	}

	if !test.closeSubscription(subscription1) {
		return
	}
	for i := 0; i < numberOfMessages; i++ {
		test.send(lastMessage, false)
		lastMessage++
	}

	if !test.closeSubscription(subscription3) {
		return
	}

	if !test.assert.Equal(numberOfMessages*3, len(*messages1)) {
		return
	}
	if !test.assert.Equal(numberOfMessages, len(*messages2)) {
		return
	}
	if !test.assert.Equal(numberOfMessages*2, len(*messages3)) {
		return
	}

	for i := 0; i < numberOfMessages*3; i++ {
		test.assert.Equal(uint32(i), (*messages1)[i])
	}
	for i := 0; i < numberOfMessages; i++ {
		test.assert.Equal(uint32(i), (*messages2)[i])
	}
	for i := 0; i < numberOfMessages*2; i++ {
		test.assert.Equal(uint32(i+numberOfMessages*2), (*messages3)[i])
	}
}

func Test_Au10_StreamReader_Errors(t *testing.T) {
	test := createStreamReaderTest(
		t,
		func(message *sarama.ConsumerMessage) (interface{}, error) {
			return nil, errors.New("test convert error")
		},
		errors.New("test error at closing"))
	defer test.close()

	errChan := make(chan error, 1)
	defer close(errChan)

	test.log.EXPECT().Debug(`Stream reading "%s" opened.`, test.getTopic())
	subscription, messages := test.createSubscription(errChan)
	if subscription == nil {
		return
	}

	test.send(uint32(1), true)
	err := <-errChan
	test.assert.EqualError(err, fmt.Sprintf(
		`failed to consume message from stream "%s": "failed to convert message from stream "%s": "test convert error""`,
		test.getTopic(), test.getTopic()))

	subscription.Close()

	if !test.assert.Equal(0, len(*messages)) {
		return
	}
}

func Test_Au10_StreamReader_StartError(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)

	log := mock_au10.NewMockLog(mock)

	service := mock_au10.NewMockService(mock)
	service.EXPECT().GetFactory().MinTimes(1).Return(factory)

	stream := au10.CreateFactory().CreateStreamReader(
		[]string{"test topic"}, nil, service)
	assert.NotNil(stream)

	service.EXPECT().Log().AnyTimes().Return(log)

	factory.EXPECT().CreateSaramaConsumer(service).
		Return(nil, errors.New("create Sarama consumer error"))

	subscription, err := stream.CreateSubscription(nil, nil)
	assert.Nil(subscription)
	assert.EqualError(err,
		`failed to start steam reading "test topic" to subscribe: "failed to open stream reading "test topic": "create Sarama consumer error""`)
}

func Test_Au10_StreamReader_NotClosedSubscription(t *testing.T) {
	test := createStreamReaderTest(t, nil, nil)
	defer test.close()
	test.log.EXPECT().Debug(`Stream reading "%s" opened.`, test.getTopic())
	test.createSubscription(nil)
	test.log.EXPECT().Error(
		`Not all (%d) subscribes of stream reading "%s" closed.`,
		1, test.getTopic())
}
