package accesspoint

import (
	fmt "fmt"

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
	ReadLog(*proto.LogReadRequest, proto.AccessPoint_ReadLogServer) error
	Post(request *proto.PostRequest) (*proto.PostResponse, error)
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

func (client *client) LogError(format string, args ...interface{}) {
	client.service.au10.Log().Error(
		client.user.GetLogin()+": "+format, args...)
}
func (client *client) LogWarn(format string, args ...interface{}) {
	client.service.au10.Log().Warn(client.user.GetLogin()+": "+format, args...)
}
func (client *client) LogInfo(format string, args ...interface{}) {
	client.service.au10.Log().Info(client.user.GetLogin()+": "+format, args...)
}
func (client *client) LogDebug(format string, args ...interface{}) {
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

func (client *client) CreateError(
	code codes.Code, descFormat string, args ...interface{}) error {

	desc := fmt.Sprintf(descFormat, args...)
	if len(desc) > 0 {
		client.LogError("Error %d: "+desc+".", code)
	}
	var externalDesc string
	if client.service.au10.Log().GetMembership().IsAvailable(
		client.user.GetRights()) {

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
	user, token, err := client.service.au10.GetUsers().Auth(request.Login)
	if err != nil {
		return nil, client.CreateError(codes.Internal, `failed to auth "%s": "%s"`,
			request.Login, err)
	}
	if token == nil {
		client.LogInfo(`Wrong creds for "%s"`, request.Login)
		return token, nil
	}
	client.user = user
	client.LogInfo(`Authed "%s".`, request.Login)
	return token, nil
}

func (client *client) ReadLog(
	request *proto.LogReadRequest, stream proto.AccessPoint_ReadLogServer) error {

	log := client.service.au10.Log()
	if err := client.checkRights(log, "read log"); err != nil {
		return err
	}
	return client.runLogSubscription(log, stream)
}

func (client *client) Post(
	request *proto.PostRequest) (*proto.PostResponse, error) {

	if request.Post != nil {
		return nil, client.CreateError(codes.InvalidArgument,
			"failed to post as post not provided")
	}
	if len(request.Post.Text) == 0 {
		return nil, client.CreateError(codes.InvalidArgument,
			"failed to post as post is empty")
	}
	posts := client.service.au10.GetPosts()
	err := client.checkRights(posts, "access posts")
	if err != nil {
		return nil, err
	}
	post := au10.CreatePostData()
	post.SetText(request.Post.Text)
	err = posts.Add(client.user, post)
	if err != nil {
		return nil, client.CreateError(codes.Internal, `failed to post: "%s"`, err)
	}
	return &proto.PostResponse{}, nil
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
	if client.service.au10.Log().GetMembership().IsAvailable(rights) {
		result = append(result, "readLog")
	}
	if client.service.au10.GetPosts().GetMembership().IsAvailable(rights) {
		result = append(result, "post")
	}
	return result
}
