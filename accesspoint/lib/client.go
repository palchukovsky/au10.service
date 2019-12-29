package accesspoint

import (
	fmt "fmt"
	"strings"

	proto "bitbucket.org/au10/service/accesspoint/proto"
	"bitbucket.org/au10/service/au10"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client is an access point connected client.
type Client interface {
	LogError(format string, args ...interface{})
	LogWarn(format string, args ...interface{})
	LogInfo(format string, args ...interface{})
	LogDebug(format string, args ...interface{})
	CreateError(code codes.Code, descFormat string, args ...interface{}) error

	GetAvailableMethods() []string

	Auth(*proto.AuthRequest) (*string, error)
	ReadLog(*proto.LogReadRequest, proto.Au10_ReadLogServer) error
	ReadPosts(*proto.PostsReadRequest, proto.Au10_ReadPostsServer) error
	ReadMessage(*proto.MessageReadRequest, proto.Au10_ReadMessageServer) error
	AddPost(request *proto.PostAddRequest) (*proto.PostAddResponse, error)
	MessageChunkWrite(*proto.MessageChunkWriteRequest) (*proto.MessageChunkWriteResponse, error)
}

// CreateClient creates new Client instance.
func CreateClient(requestID uint64, user au10.User, service Service) Client {
	return &client{service: service, requestID: requestID, user: user}
}

type client struct {
	service   Service
	requestID uint64
	user      au10.User
}

func (client *client) getLogHeader(format string) string {
	return fmt.Sprintf("%d.%s: ", client.requestID, client.user.GetLogin()) +
		format
}
func (client *client) LogError(format string, args ...interface{}) {
	client.service.GetAu10().Log().Error(client.getLogHeader(format), args...)
}
func (client *client) LogWarn(format string, args ...interface{}) {
	client.service.GetAu10().Log().Warn(client.getLogHeader(format), args...)
}
func (client *client) LogInfo(format string, args ...interface{}) {
	client.service.GetAu10().Log().Info(client.getLogHeader(format), args...)
}
func (client *client) LogDebug(format string, args ...interface{}) {
	client.service.GetAu10().Log().Debug(client.getLogHeader(format), args...)
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

func (client *client) CreateError(
	code codes.Code, descFormat string, args ...interface{}) error {

	desc := fmt.Sprintf(descFormat, args...)
	if len(desc) > 0 {
		client.LogError(strings.ToUpper(desc[:1])+desc[1:]+" (RPC error %d).", code)
	} else {
		client.LogError("RPC error %d.", code)
	}
	var externalDesc string
	if client.service.GetAu10().Log().GetMembership().IsAvailable(client.user.GetRights()) {
		externalDesc = desc
	} else {
		var ok bool
		if externalDesc, ok = starusByCode[code]; !ok {
			client.LogError("Failed to find external description for status code %d.",
				code)
			externalDesc = "unknown error"
		}
	}
	return status.Error(code, externalDesc)
}

func (client *client) Auth(request *proto.AuthRequest) (*string, error) {
	user, token, err := client.service.GetAu10().GetUsers().Auth(request.Login)
	if err != nil {
		return nil, client.CreateError(codes.Internal, `failed to auth "%s": "%s"`,
			request.Login, err)
	}
	if token == nil {
		client.LogInfo(`Wrong creds for "%s"`, request.Login)
		return nil, nil
	}
	client.user = user
	client.LogInfo(`Authed "%s".`, request.Login)
	return token, nil
}

func (client *client) ReadLog(
	request *proto.LogReadRequest, stream proto.Au10_ReadLogServer) error {

	log := client.service.GetAu10().Log()
	if err := client.checkRights(log, "read log"); err != nil {
		return err
	}
	return client.runLogSubscription(log, stream)
}

func (client *client) ReadPosts(
	*proto.PostsReadRequest, proto.Au10_ReadPostsServer) error {

	posts := client.service.GetAu10().GetPosts()
	if err := client.checkRights(posts, "read posts"); err != nil {
		return err
	}
	return client.CreateError(codes.Unimplemented, `not implemented yet`)
}

func (client *client) ReadMessage(
	*proto.MessageReadRequest, proto.Au10_ReadMessageServer) error {

	return client.CreateError(codes.Unimplemented, `not implemented yet`)
}

func (client *client) AddPost(
	request *proto.PostAddRequest) (*proto.PostAddResponse, error) {

	posts := client.service.GetAu10().GetPosts()
	err := client.checkRights(posts, "access posts")
	if err != nil {
		return nil, err
	}
	post := au10.CreatePostData()
	err = posts.Add(client.user, post)
	if err != nil {
		return nil, client.CreateError(codes.Internal, `failed to post: "%s"`, err)
	}
	return &proto.PostAddResponse{}, nil
}

func (client *client) MessageChunkWrite(
	*proto.MessageChunkWriteRequest) (*proto.MessageChunkWriteResponse, error) {

	return nil, client.CreateError(codes.Unimplemented, `not implemented yet`)
}

func (client *client) checkRights(entity au10.Member, action string) error {
	if !entity.GetMembership().IsAvailable(client.user.GetRights()) {
		return client.CreateError(codes.PermissionDenied,
			`permission denied to %s`, action)
	}
	return nil
}

func (client *client) GetAvailableMethods() []string {
	result := []string{"auth"}
	rights := client.user.GetRights()
	if client.service.GetAu10().Log().GetMembership().IsAvailable(rights) {
		result = append(result, "readLog")
	}
	if client.service.GetAu10().GetPosts().GetMembership().IsAvailable(rights) {
		result = append(result, "post")
	}
	return result
}
