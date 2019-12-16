package au10

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// Service defines Au10 service interface.
type Service interface {
	// Close closes connection to Au10 service.
	Close()

	// GetNodeType returns node type name.
	GetNodeType() string
	// GetNodeName returns node name.
	GetNodeName() string

	// GetStreamBrokers returns stream broker address list.
	GetStreamBrokers() []string

	// Log returns log service.
	Log() Log

	// GetFactory returns default objects factory.
	GetFactory() Factory

	// InitUsers inits users service to use it in the feature.
	InitUsers() error
	// GetUsers returns users service.
	GetUsers() Users

	// InitPosts inits posts service to use it in the feature.
	InitPosts() error
	// GetPosts returns posts service.
	GetPosts() Posts
}

// DialOrPanic creates a new connection to Au10 service or panics with error
// in standard output if creating is failed.
func DialOrPanic(
	nodeType, nodeName string, streamBrokers []string, factory Factory) Service {

	for i := 1; ; i++ {
		result, confErr, connErr := Dial(
			nodeType, nodeName, streamBrokers, factory)
		var err error
		if confErr != nil {
			err = confErr
		} else if connErr != nil {
			err = connErr
		} else {
			return result
		}
		logRecord := fmt.Sprintf(
			`Failed to connect to Au10 service: "%s". Stream borkers: "%s".`,
			err, streamBrokers)
		if confErr == nil && i < 5 {
			log.Println(logRecord)
			time.Sleep(factory.CreateRedialSleepTime())
		} else {
			log.Printf(logRecord)
			panic(logRecord)
		}
	}
}

// Dial creates a new connection to Au10 service.
func Dial(
	nodeType,
	nodeName string,
	streamBrokers []string,
	factory Factory) (Service, error, error) {

	if nodeType == "" {
		return nil, errors.New("node type name is empty"), nil
	}
	if nodeName == "" {
		return nil, errors.New("node name is empty"), nil
	}

	result := &service{
		nodeType:      nodeType,
		nodeName:      nodeName,
		factory:       factory,
		streamBrokers: streamBrokers}

	var err error
	result.log, err = result.factory.CreateLog(result)
	if err != nil {
		result.Close()
		return nil, nil, fmt.Errorf(`failed to open service log: "%s"`, err)
	}

	result.Log().Info(`Connected to the network. Stream brokers: "%s".`,
		strings.Join(streamBrokers, ", "))

	return result, nil, nil
}

type service struct {
	nodeType      string
	nodeName      string
	factory       Factory
	streamBrokers []string
	log           Log
	users         Users
	posts         Posts
}

func (service *service) Close() {
	if service.posts != nil {
		service.posts.Close()
	}
	if service.users != nil {
		service.users.Close()
	}
	if service.log != nil {
		service.log.Close()
	}
}

func (service *service) GetNodeType() string { return service.nodeType }
func (service *service) GetNodeName() string { return service.nodeName }
func (service *service) Log() Log            { return service.log }
func (service *service) GetFactory() Factory { return service.factory }
func (service *service) GetStreamBrokers() []string {
	return service.streamBrokers
}

func (service *service) InitUsers() error {
	if service.users != nil {
		return errors.New("users service already initiated")
	}
	var err error
	service.users, err = service.factory.CreateUsers(service.factory)
	if err != nil {
		return fmt.Errorf(`failed to create users service: "%s"`, err)
	}
	return nil
}

func (service *service) GetUsers() Users {
	if service.users == nil {
		panic("users service is not initialized")
	}
	return service.users
}

func (service *service) InitPosts() error {
	if service.posts != nil {
		return errors.New("posts service already initiated")
	}
	service.posts = service.factory.CreatePosts(service)
	return nil
}

func (service *service) GetPosts() Posts {
	if service.posts == nil {
		panic("posts service is not initialized")
	}
	return service.posts
}
