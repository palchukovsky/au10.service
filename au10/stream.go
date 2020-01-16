package au10

import "github.com/Shopify/sarama"

// NewStreamConfig creates Sarama config object.
func NewStreamConfig(service Service) *sarama.Config {
	result := sarama.NewConfig()
	result.ClientID = service.GetNodeType() + "." + service.GetNodeName()
	result.Version = sarama.V2_3_0_0
	return result
}

func (*factory) NewSaramaProducer(
	service Service,
	enableSuccesses bool) (sarama.AsyncProducer, error) {
	confing := NewStreamConfig(service)
	confing.Producer.Return.Successes = enableSuccesses
	confing.Producer.Return.Errors = true
	return sarama.NewAsyncProducer(service.GetStreamBrokers(), confing)
}

func (*factory) NewSaramaConsumer(
	service Service) (sarama.ConsumerGroup, error) {
	config := NewStreamConfig(service)
	return sarama.NewConsumerGroup(
		service.GetStreamBrokers(), config.ClientID, config)
}
