package au10

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Shopify/sarama"
)

// LogStreamTopic is a name of log stream topic.
const LogStreamTopic = "log"

// Log represents logger service.
type Log interface {
	Member

	// Close closes the service.
	Close()

	// InitSubscriptionService initiates subscriber to read logs in the feature.
	InitSubscriptionService() error
	// Subscribe creates a subscription to log-messages.
	Subscribe() (LogSubscription, error)

	// Fatal formats and sends error message in the queue and prints to the
	// standard logger and calls to os.Exit(1).
	Fatal(format string, args ...interface{})
	// Error formats and sends error message in the queue and prints to the
	// standard logger.
	Error(format string, args ...interface{})
	// Warn formats and sends warning message in the queue and prints to the
	// standard logger.
	Warn(format string, args ...interface{})
	// Info formats and sends information message in the queue and prints to the
	// standard logger.
	Info(format string, args ...interface{})
	// Debug formats and sends debug message in the queue and prints to the
	// standard logger.
	Debug(format string, args ...interface{})
}

// LogSubscription represents subscription to logger data.
type LogSubscription interface {
	Subscription
	// GetRecordsChan resturns incoming records channel.
	GetRecordsChan() <-chan LogRecord
}

// LogRecord represents log-record delivered from a queue.
type LogRecord interface {
	// GetSequenceNumber returns record sequence number.
	GetSequenceNumber() int64
	// GetTime returns record time.
	GetTime() time.Time
	// GetText returns record text.
	GetText() string
	// GetSeverity returns record severity.
	GetSeverity() string
	// GetNodeType returns author node type name.
	GetNodeType() string
	// GetNodeName returns author node instance name.
	GetNodeName() string
}

////////////////////////////////////////////////////////////////////////////////

func (*factory) CreateLog(service Service) (Log, error) {
	writer, err := service.GetFactory().CreateStreamWriter(LogStreamTopic, service)
	if err != nil {
		return nil, err
	}
	return &logService{
			service:    service,
			membership: CreateMembership("", ""),
			writer:     writer},
		nil
}

type logService struct {
	service    Service
	membership Membership
	writer     StreamWriter
	reader     StreamReader
}

func (service *logService) Close() {
	if service.reader != nil {
		service.reader.Close()
	}
	service.writer.Close()
}

func (service *logService) GetMembership() Membership {
	return service.membership
}

func (service *logService) InitSubscriptionService() error {
	if service.reader != nil {
		return errors.New("log subscription service already initiated")
	}
	service.reader = service.service.GetFactory().CreateStreamReader(
		[]string{LogStreamTopic},
		func(source *sarama.ConsumerMessage) (interface{}, error) {
			return CreateLogRecord(source)
		},
		service.service)
	return nil
}

func (service *logService) Subscribe() (LogSubscription, error) {
	if service.reader == nil {
		panic("log subscription service is not initialized")
	}
	return createLogSubscription(service)
}

func (service *logService) Fatal(format string, args ...interface{}) {
	service.publish(4, format, args...)
	panic(fmt.Sprintf(format, args...))
}

func (service *logService) Error(format string, args ...interface{}) {
	service.publish(3, format, args...)
}

func (service *logService) Warn(format string, args ...interface{}) {
	service.publish(2, format, args...)
}

func (service *logService) Info(format string, args ...interface{}) {
	service.publish(1, format, args...)
}

func (service *logService) Debug(format string, args ...interface{}) {
	service.publish(0, format, args...)
}

func (service *logService) publish(
	severity uint8, format string, args ...interface{}) {

	message := fmt.Sprintf(format, args...)
	log.Print(resolveLogSeverity(severity) + ":\t" + message)
	err := service.writer.Push(LogRecordData{
		Node:     service.service.GetNodeName(),
		Severity: severity,
		Text:     message})
	if err != nil {
		log.Printf(resolveLogSeverity(3)+":\t"+`Failed to store log-record: "%s".`,
			err)
	}
}

func resolveLogSeverity(severity uint8) string {
	switch severity {
	case 0:
		return "debug"
	case 1:
		return "info"
	case 2:
		return "warn"
	default:
		return "error"
	case 4:
		return "fatal"
	}
}

////////////////////////////////////////////////////////////////////////////////

type logSubscription struct {
	subscription
	stream      StreamSubscription
	recordsChan chan LogRecord
}

func createLogSubscription(service *logService) (LogSubscription, error) {
	result := &logSubscription{
		subscription: createSubscription(),
		recordsChan:  make(chan LogRecord, 1)}

	var err error
	result.stream, err = service.reader.CreateSubscription(
		func(message interface{}) {
			result.recordsChan <- message.(LogRecord)
		},
		result.errChan)
	if err != nil {
		result.Close()
		return nil, err
	}
	return result, nil
}

func (subscription *logSubscription) Close() {
	if subscription.stream != nil {
		subscription.stream.Close()
	}
	close(subscription.recordsChan)
	subscription.subscription.close()
}

func (subscription *logSubscription) GetRecordsChan() <-chan LogRecord {
	return subscription.recordsChan
}

func (subscription *logSubscription) GetErrChan() <-chan error {
	return subscription.errChan
}

////////////////////////////////////////////////////////////////////////////////

// LogRecordData describes log record in a stream.
type LogRecordData struct {
	Node     string
	Severity uint8
	Text     string
}

type logRecord struct {
	message *sarama.ConsumerMessage
	data    LogRecordData
}

func (message *logRecord) GetSequenceNumber() int64 {
	return message.message.Offset
}
func (message *logRecord) GetTime() time.Time {
	return message.message.Timestamp
}
func (message *logRecord) GetText() string {
	return message.data.Text
}
func (message *logRecord) GetSeverity() string {
	return resolveLogSeverity(message.data.Severity)
}
func (message *logRecord) GetNodeType() string {
	return string(message.message.Key)
}
func (message *logRecord) GetNodeName() string {
	return message.data.Node
}

// CreateLogRecord creates new LogRecord from stream data.
func CreateLogRecord(source *sarama.ConsumerMessage) (LogRecord, error) {
	result := &logRecord{message: source}
	if err := json.Unmarshal(source.Value, &result.data); err != nil {
		return nil, fmt.Errorf(`failed to parse log-record: "%s"`, err)
	}
	return result, nil
}

////////////////////////////////////////////////////////////////////////////////
