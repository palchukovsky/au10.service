package au10

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/Shopify/sarama"
)

// StreamWriter describes interface to write data to a stream.
type StreamWriter interface {
	Close()
	Push(interface{}) error
}

func (*factory) NewStreamWriter(
	topic string, service Service) (StreamWriter, error) {

	result := &streamWriter{
		service: service,
		topic:   topic,
		key:     sarama.StringEncoder(service.GetNodeType())}

	var err error
	result.producer, err = service.GetFactory().NewSaramaProducer(service)
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
				result.service.Log().Error(logMessage)
			}
		}
		result.stopBarrier.Done()
	}()

	return result, nil
}

type streamWriter struct {
	service     Service
	topic       string
	key         sarama.Encoder
	producer    sarama.AsyncProducer
	stopBarrier sync.WaitGroup
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
			stream.service.Log().Error(message)
		}
	}
	stream.stopBarrier.Wait()
}

func (stream *streamWriter) Push(source interface{}) error {
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
