package au10

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Shopify/sarama"
)

// Log represents logger service.
type Log interface {
	Member

	// Close closes the service.
	Close()

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

// LogReader represents log recordes service service.
type LogReader interface {
	Member

	// Close closes the service.
	Close()

	// Subscribe creates a subscription to log-messages.
	Subscribe() (LogSubscription, error)
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

const logStreamTopic = "log"

func (*factory) NewLog(service Service) (Log, error) {
	stream, err := service.GetFactory().NewStreamWriter(logStreamTopic,
		service)
	if err != nil {
		return nil, err
	}
	return &logService{
			service:    service,
			membership: NewMembership("", ""),
			stream:     stream},
		nil
}

type logService struct {
	service    Service
	membership Membership
	stream     StreamWriter
}

func (service *logService) Close() {
	service.stream.Close()
}

func (service *logService) GetMembership() Membership {
	return service.membership
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
	err := service.stream.Push(LogRecordData{
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

// NewLogReader creates log reader instance.
func NewLogReader(service Service) LogReader {
	return &logReader{
		membership: NewMembership("", ""),
		stream: service.GetFactory().NewStreamReader(
			[]string{logStreamTopic},
			func(source *sarama.ConsumerMessage) (interface{}, error) {
				return ConvertSaramaMessageIntoLogRecord(source)
			},
			service)}
}

func (reader *logReader) Close() {
	reader.stream.Close()
}

func (reader *logReader) GetMembership() Membership { return reader.membership }

func (reader *logReader) Subscribe() (LogSubscription, error) {
	result := &logSubscription{
		subscription: newSubscription(),
		recordsChan:  make(chan LogRecord, 1)}
	var err error
	result.stream, err = reader.stream.NewSubscription(
		func(message interface{}) { result.recordsChan <- message.(LogRecord) },
		result.errChan)
	if err != nil {
		result.Close()
		return nil, err
	}
	return result, nil
}

type logReader struct {
	membership Membership
	stream     StreamReader
}

type logSubscription struct {
	subscription
	stream      StreamSubscription
	recordsChan chan LogRecord
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
	Node     string `json:"n"`
	Severity uint8  `json:"s"`
	Text     string `json:"t"`
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

// ConvertSaramaMessageIntoLogRecord creates new LogRecord from stream data.
func ConvertSaramaMessageIntoLogRecord(
	source *sarama.ConsumerMessage) (LogRecord, error) {

	result := &logRecord{message: source}
	if err := json.Unmarshal(source.Value, &result.data); err != nil {
		return nil, fmt.Errorf(`failed to parse log-record: "%s"`, err)
	}
	return result, nil
}

////////////////////////////////////////////////////////////////////////////////
