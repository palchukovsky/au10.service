package accesspoint

import (
	fmt "fmt"
	"strings"

	"bitbucket.org/au10/service/au10"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// Client is an access point connected client.
type Client interface {
	Auth(*AuthRequest) (*AuthResponse, error)
	SubscribeToLog(*LogReadRequest, AccessPoint_ReadLogServer) error
}

// CreateClient creates new Client instance.
func CreateClient(user au10.User, service *service) Client {
	return &client{
		service: service,
		user:    user}
}

type client struct {
	service *service
	user    au10.User
}

func (client *client) logError(format string, args ...interface{}) {
	client.service.au10.Log().Error(
		client.user.GetLogin()+": "+format, args...)
}
func (client *client) logWarn(format string, args ...interface{}) {
	client.service.au10.Log().Warn(client.user.GetLogin()+": "+format, args...)
}
func (client *client) logInfo(format string, args ...interface{}) {
	client.service.au10.Log().Info(client.user.GetLogin()+": "+format, args...)
}
func (client *client) logDebug(format string, args ...interface{}) {
	client.service.au10.Log().Debug(
		client.user.GetLogin()+": "+format, args...)
}

var starusByCode = map[codes.Code]string{
	codes.OK:                 "OK",
	codes.Canceled:           "CANCELLED",
	codes.Unknown:            "UNKNOWN",
	codes.InvalidArgument:    "INVALID_ARGUMENT",
	codes.DeadlineExceeded:   "DEADLINE_EXCEEDED",
	codes.NotFound:           "NOT_FOUND",
	codes.AlreadyExists:      "ALREADY_EXISTS",
	codes.PermissionDenied:   "PERMISSION_DENIED",
	codes.ResourceExhausted:  "RESOURCE_EXHAUSTED",
	codes.FailedPrecondition: "FAILED_PRECONDITION",
	codes.Aborted:            "ABORTED",
	codes.OutOfRange:         "OUT_OF_RANGE",
	codes.Unimplemented:      "UNIMPLEMENTED",
	codes.Internal:           "INTERNAL",
	codes.Unavailable:        "UNAVAILABLE",
	codes.DataLoss:           "DATA_LOSS",
	codes.Unauthenticated:    "UNAUTHENTICATED",
}

func (client *client) createError(
	code codes.Code, descFormat string, args ...interface{}) error {

	desc := fmt.Sprintf(descFormat, args...)
	if len(desc) > 0 {
		client.logError(strings.ToUpper(desc[0:1]) + desc[1:] + ".")
	}
	var externalDesc string
	if client.service.au10.Log().GetMembership().IsAvailable(
		client.user.GetRights()) {

		externalDesc = desc
	} else {
		var ok bool
		if externalDesc, ok = starusByCode[code]; !ok {
			client.logError("Failed to find external description for status code %d.",
				code)
			externalDesc = "unknown error"
		}
	}
	return status.Error(code, externalDesc)
}

func (client *client) Auth(request *AuthRequest) (*AuthResponse, error) {
	return nil, client.createError(
		codes.Unimplemented,
		`called method without implementation "auth" for login "%s"`, request.Login)
}

func (client *client) SubscribeToLog(
	request *LogReadRequest, stream AccessPoint_ReadLogServer) error {

	return client.runLogSubscription(stream)
}
