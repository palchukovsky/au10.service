package au10

import "github.com/Shopify/sarama"

// NewStreamConfig creates Sarama config object.
func NewStreamConfig(service Service) *sarama.Config {
	result := sarama.NewConfig()
	result.ClientID = service.GetNodeType() + "." + service.GetNodeName()
	result.Version = sarama.V2_3_0_0
	result.Producer.Return.Errors = true
	return result
}

func (*factory) NewSaramaProducer(
	service Service) (sarama.AsyncProducer, error) {

	return sarama.NewAsyncProducer(
		service.GetStreamBrokers(), NewStreamConfig(service))
}

func (*factory) NewSaramaConsumer(
	service Service) (sarama.ConsumerGroup, error) {

	config := NewStreamConfig(service)
	return sarama.NewConsumerGroup(
		service.GetStreamBrokers(), config.ClientID, config)
}
