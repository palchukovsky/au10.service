package au10

import (
	"time"

	"github.com/Shopify/sarama"
)

// Factory provides the interface to create Au10 service object instances.
type Factory interface {
	CreateRedialSleepTime() time.Duration

	CreateStreamReader(
		topics []string,
		convertMessage func(*sarama.ConsumerMessage) (interface{}, error),
		service Service) StreamReader
	CreateStreamWriter(topic string, service Service) (StreamWriter, error)

	CreateLog(service Service) (Log, error)

	CreateUsers(Factory) (Users, error)
	CreateUser(login string, membership Membership, rights []Rights) (User, error)

	CreateSaramaProducer(service Service) (sarama.AsyncProducer, error)
	CreateSaramaConsumer(service Service) (sarama.ConsumerGroup, error)
}

// CreateFactory creates an instance of Factory interface.
func CreateFactory() Factory { return new(factory) }

type factory struct{}

func (*factory) CreateRedialSleepTime() time.Duration { return 3 * time.Second }
