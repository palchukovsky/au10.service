package au10

import "github.com/Shopify/sarama"

// CreateStreamConfig creates Sarama config object.
func CreateStreamConfig(service Service) *sarama.Config {
	result := sarama.NewConfig()
	result.ClientID = service.GetNodeType() + "." + service.GetNodeName()
	result.Version = sarama.V2_3_0_0
	result.Producer.Return.Errors = true
	return result
}

func (*factory) CreateSaramaProducer(
	service Service) (sarama.AsyncProducer, error) {

	return sarama.NewAsyncProducer(
		service.GetStreamBrokers(), CreateStreamConfig(service))
}

func (*factory) CreateSaramaConsumer(
	service Service) (sarama.ConsumerGroup, error) {

	config := CreateStreamConfig(service)
	return sarama.NewConsumerGroup(
		service.GetStreamBrokers(), config.ClientID, config)
}
