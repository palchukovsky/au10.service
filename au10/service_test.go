package au10_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_Service_DialOrPanic(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)

	brokers := []string{"broker 1", "broker 2"}

	assert.Panics(func() {
		au10.DialOrPanic("", "test node name", brokers, factory)
	})
	assert.Panics(func() {
		au10.DialOrPanic("test node type", "", brokers, factory)
	})

	var serviceArg au10.Service
	numberOfAttempts := 5
	factory.EXPECT().NewLog(gomock.Any()).
		Times(numberOfAttempts).
		Do(func(service au10.Service) { serviceArg = service }).
		Return(nil, errors.New("test error"))
	factory.EXPECT().NewRedialSleepTime().Times(numberOfAttempts - 1).
		Return(0 * time.Second)
	assert.Panics(func() {
		au10.DialOrPanic("test node type", "test node name", brokers, factory)
	})

	log := mock_au10.NewMockLog(mock)
	log.EXPECT().Close()
	log.EXPECT().Info(`Connected to the network. Stream brokers: "%s".`,
		strings.Join(brokers, ", "))
	factory.EXPECT().NewLog(gomock.Any()).
		Do(func(service au10.Service) { serviceArg = service }).
		Return(log, nil)
	var service au10.Service
	assert.NotPanics(func() {
		service = au10.DialOrPanic(
			"test node type", "test node name", brokers, factory)
	})
	assert.NotNil(service)
	assert.True(service == serviceArg)
	service.Close()
}

func Test_Au10_Service_Deal(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	factory := mock_au10.NewMockFactory(mock)

	brokers := []string{"broker 1", "broker 2"}

	service, confErr, connErr := au10.Dial(
		"", "test node name", brokers, factory)
	assert.Nil(service)
	assert.EqualError(confErr, "node type name is empty")
	assert.NoError(connErr)

	service, confErr, connErr = au10.Dial(
		"test node type", "", brokers, factory)
	assert.Nil(service)
	assert.EqualError(confErr, "node name is empty")
	assert.NoError(connErr)

	factory.EXPECT().NewLog(gomock.Any()).
		Return(nil, errors.New("test error"))
	service, confErr, connErr = au10.Dial(
		"test node type", "test node name", brokers, factory)
	assert.Nil(service)
	assert.NoError(confErr)
	assert.EqualError(connErr, `failed to open service log: "test error"`)

	log := mock_au10.NewMockLog(mock)
	log.EXPECT().Close()
	log.EXPECT().Info(`Connected to the network. Stream brokers: "%s".`,
		strings.Join(brokers, ", "))
	var serviceArg au10.Service
	factory.EXPECT().NewLog(gomock.Any()).Return(log, nil).
		Do(func(service au10.Service) { serviceArg = service })
	service, confErr, connErr = au10.Dial(
		"test node type", "test node name", brokers, factory)
	assert.NotNil(service)
	assert.True(service == serviceArg)
	assert.NoError(confErr)
	assert.NoError(connErr)

	assert.Equal("test node type", service.GetNodeType())
	assert.Equal("test node name", service.GetNodeName())
	assert.Equal(brokers, service.GetStreamBrokers())
	assert.True(service.Log() == log)
	assert.True(service.GetFactory() == factory)

	service.Close()
}
