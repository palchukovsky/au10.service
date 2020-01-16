package au10

import (
	"time"

	"github.com/Shopify/sarama"
)

// Factory provides the interface to create Au10 service object instances.
type Factory interface {
	NewRedialSleepTime() time.Duration

	NewStreamReader(
		topics []string,
		convertMessage func(*sarama.ConsumerMessage) (interface{}, error),
		service Service) StreamReader
	NewStreamWriter(topic string, service Service) (StreamWriter, error)
	NewStreamWriterWithResult(
		topic string,
		service Service) (StreamWriterWithResult, error)

	NewLog(service Service) (Log, error)

	NewUser(
		id UserID,
		login string,
		membership Membership,
		rights []Rights) (User, error)

	NewSaramaProducer(
		service Service,
		enableSuccess bool) (sarama.AsyncProducer, error)
	NewSaramaConsumer(Service) (sarama.ConsumerGroup, error)
}

// NewFactory creates an instance of Factory interface.
func NewFactory() Factory { return new(factory) }

type factory struct{}

func (*factory) NewRedialSleepTime() time.Duration { return 3 * time.Second }
