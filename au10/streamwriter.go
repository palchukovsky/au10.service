package au10

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Shopify/sarama"
)

// StreamWriter describes the interface to write data to a stream.
type StreamWriter interface {
	Close()
	PushAsync(interface{}) error
}

func (*factory) NewStreamWriter(
	topic string,
	service Service) (StreamWriter, error) {

	result, err := newStreamWriter(topic, false, service)
	if err != nil {
		return nil, err
	}

	result.stopBarrier.Add(1)
	go func() {
		for errMessage := range result.producer.Errors() {
			logMessage := fmt.Sprintf(`Stream "%s" writing error: "%s".`,
				result.topic, errMessage.Err)
			if result.topic == logStreamTopic {
				// for log stream, this record creates a sequence of calls without end
				log.Printf(logMessage)
			} else {
				result.log.Error(logMessage)
			}
		}
		result.stopBarrier.Done()
	}()

	return result, nil
}

// StreamWriterWithResult describes the interface to write data to a stream
// and wait for result.
type StreamWriterWithResult interface {
	Close()
	Push(interface{}) (id uint32, timestamp time.Time, err error)
}

func (*factory) NewStreamWriterWithResult(
	topic string,
	service Service) (StreamWriterWithResult, error) {
	return newStreamWriter(topic, true, service)
}

type streamWriter struct {
	log         Log
	topic       string
	key         sarama.Encoder
	producer    sarama.AsyncProducer
	stopBarrier sync.WaitGroup
}

func newStreamWriter(
	topic string,
	enableSuccess bool,
	service Service) (*streamWriter, error) {
	result := &streamWriter{
		log:   service.Log(),
		topic: topic,
		key:   sarama.StringEncoder(service.GetNodeType())}
	var err error
	result.producer, err = service.GetFactory().NewSaramaProducer(
		service, enableSuccess)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (stream *streamWriter) Close() {
	err := stream.producer.Close()
	if err != nil {
		message := fmt.Sprintf(`Failed to close stream writing "%s": "%s".`,
			stream.topic, err)
		if stream.topic == logStreamTopic {
			// for log stream, this record creates a sequence of calls without end
			log.Printf(message)
		} else {
			stream.log.Error(message)
		}
	}
	stream.stopBarrier.Wait()
}

func (stream *streamWriter) PushAsync(source interface{}) error {
	serialized, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf(`failed to serialize: "%s"`, err)
	}
	stream.producer.Input() <- &sarama.ProducerMessage{
		Topic: stream.topic,
		Key:   stream.key,
		Value: sarama.ByteEncoder(serialized)}
	return nil
}

func (stream *streamWriter) Push(
	source interface{}) (id uint32, timestamp time.Time, err error) {

	serialized, err := json.Marshal(source)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf(`failed to serialize: "%s"`, err)
	}
	stream.producer.Input() <- &sarama.ProducerMessage{
		Topic: stream.topic,
		Key:   stream.key,
		Value: sarama.ByteEncoder(serialized)}

	select {
	case message, isOpened := <-stream.producer.Errors():
		var err error
		if !isOpened {
			err = errors.New("stream closed")
		} else {
			err = message.Err
		}
		return 0, time.Time{}, err
	case message, isOpened := <-stream.producer.Successes():
		if !isOpened {
			return 0, time.Time{}, errors.New("stream closed")
		}
		return uint32(message.Offset), message.Timestamp, nil
	}
}
