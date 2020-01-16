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
	RegisterError(code codes.Code, descFormat string, args ...interface{}) error

	GetAllowedMethods() *proto.AuthResponse_AllowedMethods

	Auth(*proto.AuthRequest) (*string, error)
	ReadLog(*proto.LogReadRequest, proto.Au10_ReadLogServer) error
	ReadPosts(*proto.PostsReadRequest, proto.Au10_ReadPostsServer) error
	ReadMessage(*proto.MessageReadRequest, proto.Au10_ReadMessageServer) error
	AddVocal(request *proto.VocalAddRequest) (*proto.Vocal, error)
	WriteMessageChunk(*proto.MessageChunkWriteRequest) (*proto.MessageChunkWriteResponse, error)
}

// NewClient creates new Client instance.
func NewClient(
	requestID uint64, request string, user au10.User, service Service) Client {

	return &client{
		service:   service,
		request:   request,
		requestID: requestID,
		user:      user}
}

type client struct {
	service   Service
	requestID uint64
	request   string
	user      au10.User
}

func (client *client) getLogHeader(format string) string {
	return fmt.Sprintf("[%s.%d.%d] ",
		client.request, client.requestID, client.user.GetID()) +
		format
}
func (client *client) LogError(format string, args ...interface{}) {
	client.service.Log().Error(client.getLogHeader(format), args...)
}
func (client *client) LogWarn(format string, args ...interface{}) {
	client.service.Log().Warn(client.getLogHeader(format), args...)
}
func (client *client) LogInfo(format string, args ...interface{}) {
	client.service.Log().Info(client.getLogHeader(format), args...)
}
func (client *client) LogDebug(format string, args ...interface{}) {
	client.service.Log().Debug(client.getLogHeader(format), args...)
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

func (client *client) RegisterError(
	code codes.Code, descFormat string, args ...interface{}) error {
	desc := fmt.Sprintf(descFormat, args...)
	if len(desc) > 0 {
		client.LogError(strings.ToUpper(desc[:1])+desc[1:]+" (RPC error %d).", code)
	} else {
		client.LogError("RPC error %d.", code)
	}
	var externalDesc string
	if client.service.Log().GetMembership().IsAllowed(client.user.GetRights()) {
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
	user, token, err := client.service.GetUsers().Auth(request.Login)
	if err != nil {
		return nil, client.RegisterError(codes.Internal, `failed to auth "%s": "%s"`,
			request.Login, err)
	}
	if token == nil {
		client.LogInfo(`Wrong creds for "%s"`, request.Login)
		return nil, nil
	}
	prevUser := client.user
	client.user = user
	client.LogInfo(`Authed "%s" (was %d).`, request.Login, prevUser.GetID())
	return token, nil
}

func (client *client) ReadLog(
	request *proto.LogReadRequest, stream proto.Au10_ReadLogServer) error {
	log := client.service.GetLogReader()
	if err := client.checkRights(log, "log"); err != nil {
		return err
	}
	return client.runLogSubscription(log, stream)
}

func (client *client) ReadPosts(
	request *proto.PostsReadRequest, stream proto.Au10_ReadPostsServer) error {
	posts := client.service.GetPosts()
	if err := client.checkRights(posts, "posts"); err != nil {
		return err
	}
	return client.runPostsSubscription(posts, stream)
}

func (client *client) ReadMessage(
	request *proto.MessageReadRequest,
	stream proto.Au10_ReadMessageServer) error {
	message, err := client.getMessage(request.PostID, request.MessageID)
	if err != nil {
		return err
	}
	chunkSize := client.service.GetGlobalProps().MaxChunkSize
	chunk := &proto.MessageChunk{Chunk: make([]byte, chunkSize)}
	messageSize := message.GetSize()
	for offset := uint64(0); offset <= messageSize; offset += chunkSize {
		if err := message.Load(&chunk.Chunk, offset); err != nil {
			return client.RegisterError(codes.Internal,
				`failed to load chunk "%s"/"%s"/%d: "%s"`,
				request.PostID, request.MessageID, offset, err)
		}
		if err := stream.Send(chunk); err != nil {
			return client.RegisterError(codes.Internal,
				`failed to send chunk "%s"/"%s"/%d: "%s"`,
				request.PostID, request.MessageID, offset, err)
		}
	}
	return nil
}

func (client *client) AddVocal(
	request *proto.VocalAddRequest) (*proto.Vocal, error) {

	var err error
	vocalRequest := covertVocalDeclarationFromProto(request, client.user, &err)
	if err != nil {
		return nil, client.RegisterError(codes.Internal,
			`failed to convert message kind: "%s"`, err)
	}

	publisher := client.service.GetPublisher()
	err = client.checkRights(publisher, "publishing")
	if err != nil {
		return nil, err
	}

	var result au10.Vocal
	result, err = publisher.PublishVocal(vocalRequest)
	if err != nil {
		return nil, client.RegisterError(codes.Internal, `failed to add: "%s"`, err)
	}

	response := convertVocalToProto(result, &err)
	if err != nil {
		return nil, client.RegisterError(codes.Internal,
			`failed to convert vocal: "%s"`, err)
	}
	return response, nil
}

func (client *client) WriteMessageChunk(
	request *proto.MessageChunkWriteRequest) (*proto.MessageChunkWriteResponse, error) {
	message, err := client.getMessage(request.PostID, request.MessageID)
	if err != nil {
		return nil, err
	}
	if err = message.Append(request.Chunk); err != nil {
		return nil, client.RegisterError(codes.Internal,
			`failed to write chunk "%s"/"%s": "%s"`,
			request.PostID, request.MessageID, err)
	}
	return &proto.MessageChunkWriteResponse{}, nil
}

func (client *client) getMessage(
	requestPostID, requestMessageID string) (au10.Message, error) {

	var err error
	intPostID := convertIDFromProto(requestPostID, &err)
	if err != nil {
		return nil, client.RegisterError(codes.InvalidArgument,
			`failed to parse post ID "%s"/"%s": "%s"`,
			requestPostID, requestMessageID, err)
	}
	intMessageID := convertIDFromProto(requestMessageID, &err)
	if err != nil {
		return nil, client.RegisterError(codes.InvalidArgument,
			`failed to parse message ID "%s"/"%s": "%s"`,
			requestPostID, requestMessageID, err)
	}

	posts := client.service.GetPosts()
	if err := client.checkRights(posts, "posts"); err != nil {
		return nil, err
	}

	var post au10.Post
	if post, err = posts.GetPost(au10.PostID(intPostID)); err != nil {
		return nil, client.RegisterError(codes.InvalidArgument,
			`failed to get post "%s"/"%s": "%s"`,
			requestPostID, requestMessageID, err)
	}
	err = client.checkRights(post, `post "%s"/"%s"`,
		requestPostID, requestMessageID)
	if err != nil {
		return nil, err
	}
	messageID := au10.MessageID(intMessageID)
	var message au10.Message
	for _, m := range post.GetMessages() {
		if m.GetID() == messageID {
			message = m
			break
		}
	}
	if message == nil {
		return nil, client.RegisterError(codes.InvalidArgument,
			`failed to find message "%s"/"%s"`, requestPostID, requestMessageID)
	}
	err = client.checkRights(message, `message "%s"/"%s"`,
		requestPostID, requestMessageID)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (client *client) checkRights(
	entity au10.Member,
	format string,
	args ...interface{}) error {
	if !entity.GetMembership().IsAllowed(client.user.GetRights()) {
		return client.RegisterError(codes.PermissionDenied,
			"Permission denied for "+format, args...)
	}
	return nil
}

func (client *client) GetAllowedMethods() *proto.AuthResponse_AllowedMethods {
	rights := client.user.GetRights()
	return &proto.AuthResponse_AllowedMethods{
		ReadLog:   client.service.Log().GetMembership().IsAllowed(rights),
		ReadPosts: client.service.GetPosts().GetMembership().IsAllowed(rights),
		AddVocal:  client.service.GetPublisher().GetMembership().IsAllowed(rights)}
}
