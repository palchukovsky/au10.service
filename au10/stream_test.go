package au10_test

import (
	"testing"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	"github.com/Shopify/sarama"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_StreamWriter_NewStreamConfig(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)

	service := mock_au10.NewMockService(ctrl)
	service.EXPECT().GetNodeType().Return("test node type")
	service.EXPECT().GetNodeName().Return("test node name")

	config := au10.NewStreamConfig(service)
	assert.NotNil(config)
	assert.Equal("test node type.test node name", config.ClientID)
	assert.Equal(sarama.V2_3_0_0, config.Version)
	assert.True(config.Producer.Return.Errors)
}

func Test_Au10_StreamWriter_NewSaramaProducer(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)

	service := mock_au10.NewMockService(ctrl)
	service.EXPECT().GetNodeType().Return("test node type")
	service.EXPECT().GetNodeName().Return("test node name")
	service.EXPECT().GetStreamBrokers().Return([]string{})

	producer, err := au10.NewFactory().NewSaramaProducer(service, false)
	assert.Nil(producer)
	assert.NotNil(err)
}

func Test_Au10_StreamWriter_NewSaramaConsumer(test *testing.T) {
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	assert := assert.New(test)

	service := mock_au10.NewMockService(ctrl)
	service.EXPECT().GetNodeType().Return("test node type")
	service.EXPECT().GetNodeName().Return("test node name")
	service.EXPECT().GetStreamBrokers().Return([]string{})

	consumer, err := au10.NewFactory().NewSaramaConsumer(service)
	assert.Nil(consumer)
	assert.NotNil(err)
}
