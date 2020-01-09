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
			time.Sleep(factory.NewRedialSleepTime())
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
	result.log, err = result.factory.NewLog(result)
	if err != nil {
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
}

func (service *service) Close() {
	service.log.Close()
}

func (service *service) GetNodeType() string { return service.nodeType }
func (service *service) GetNodeName() string { return service.nodeName }
func (service *service) Log() Log            { return service.log }
func (service *service) GetFactory() Factory { return service.factory }
func (service *service) GetStreamBrokers() []string {
	return service.streamBrokers
}
