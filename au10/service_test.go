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
	factory.EXPECT().CreateLog(gomock.Any()).
		Times(numberOfAttempts).
		Do(func(service au10.Service) { serviceArg = service }).
		Return(nil, errors.New("test error"))
	factory.EXPECT().CreateRedialSleepTime().Times(numberOfAttempts - 1).
		Return(0 * time.Second)
	assert.Panics(func() {
		au10.DialOrPanic("test node type", "test node name", brokers, factory)
	})

	log := mock_au10.NewMockLog(mock)
	log.EXPECT().Close()
	log.EXPECT().Info(`Connected to the network. Stream brokers: "%s".`,
		strings.Join(brokers, ", "))
	factory.EXPECT().CreateLog(gomock.Any()).
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

	factory.EXPECT().CreateLog(gomock.Any()).
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
	factory.EXPECT().CreateLog(gomock.Any()).Return(log, nil).
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

type serviceTest struct {
	test    *testing.T
	mock    *gomock.Controller
	assert  *assert.Assertions
	factory *mock_au10.MockFactory
	log     *mock_au10.MockLog
	service au10.Service
}

func createServiceTest(test *testing.T) *serviceTest {
	result := &serviceTest{
		test:   test,
		mock:   gomock.NewController(test),
		assert: assert.New(test)}

	brokers := []string{"broker 1", "broker 2"}

	result.log = mock_au10.NewMockLog(result.mock)
	result.log.EXPECT().Close()
	result.log.EXPECT().Info(`Connected to the network. Stream brokers: "%s".`,
		strings.Join(brokers, ", "))

	result.factory = mock_au10.NewMockFactory(result.mock)
	result.factory.EXPECT().CreateLog(gomock.Any()).Return(result.log, nil)

	var confErr error
	var connErr error
	result.service, confErr, connErr = au10.Dial(
		"test node type", "test node", brokers, result.factory)
	result.assert.NoError(confErr)
	result.assert.NoError(connErr)
	result.assert.NotNil(result.service)

	return result
}

func (test *serviceTest) close() {
	test.service.Close()
	test.mock.Finish()
}

func Test_Au10_Service_Users(t *testing.T) {
	test := createServiceTest(t)
	defer test.close()

	test.assert.PanicsWithValue("users service is not initialized", func() {
		test.service.GetUsers()
	})

	test.factory.EXPECT().CreateUsers(test.factory).
		Return(nil, errors.New("test users error"))
	test.assert.EqualError(test.service.InitUsers(),
		`failed to create users service: "test users error"`)

	users := mock_au10.NewMockUsers(test.mock)
	users.EXPECT().Close()
	test.factory.EXPECT().CreateUsers(test.factory).Return(users, nil)
	test.assert.NoError(test.service.InitUsers())

	test.assert.EqualError(test.service.InitUsers(),
		"users service already initiated")

	var result au10.Users
	test.assert.NotPanics(func() { result = test.service.GetUsers() })
	test.assert.True(result == users)
}
