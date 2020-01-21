package accesspoint

import (
	"context"
	"sync/atomic"

	proto "bitbucket.org/au10/service/accesspoint/proto"
	"bitbucket.org/au10/service/au10"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Service describes access point service interface.
type Service interface {
	GetGlobalProps() *proto.Props

	Log() au10.Log
	GetLogReader() au10.LogReader

	GetPosts() au10.Posts
	GetPublisher() au10.Publisher

	GetUsers() au10.Users

	GetLogSubscriptionInfo() *SubscriptionInfo
	GetPostsSubscriptionInfo() *SubscriptionInfo
	RegisterSubscriber() uint32
	UnregisterSubscriber() uint32
}

type service struct {
	service au10.Service

	props *proto.Props

	log       au10.Log
	logReader au10.LogReader
	posts     au10.Posts
	publisher au10.Publisher
	users     au10.Users

	defaultUser au10.User
	newClient   func(
		requestID uint32,
		request string,
		user au10.User,
		service Service) Client
	grpc Grpc

	lastRequestID         uint32
	numberOfSubscribers   uint32
	logSubscriptionInfo   *SubscriptionInfo
	postsSubscriptionInfo *SubscriptionInfo
}

// NewAu10Server creates Au10Server server implementation instance.
func NewAu10Server(
	props *proto.Props,
	log au10.Log,
	logReader au10.LogReader,
	posts au10.Posts,
	publisher au10.Publisher,
	users au10.Users,
	defaultUser au10.User,
	newClient func(
		requestID uint32,
		request string,
		user au10.User,
		service Service) Client,
	grpc Grpc) proto.Au10Server {
	return &service{
		props:                 props,
		log:                   log,
		logReader:             logReader,
		posts:                 posts,
		publisher:             publisher,
		users:                 users,
		defaultUser:           defaultUser,
		newClient:             newClient,
		grpc:                  grpc,
		logSubscriptionInfo:   &SubscriptionInfo{},
		postsSubscriptionInfo: &SubscriptionInfo{}}
}

func (service *service) Log() au10.Log        { return service.log }
func (service *service) GetPosts() au10.Posts { return service.posts }
func (service *service) GetUsers() au10.Users { return service.users }
func (service *service) GetLogReader() au10.LogReader {
	return service.logReader
}
func (service *service) GetPublisher() au10.Publisher {
	return service.publisher
}

func (service *service) GetLogSubscriptionInfo() *SubscriptionInfo {
	return service.logSubscriptionInfo
}
func (service *service) GetPostsSubscriptionInfo() *SubscriptionInfo {
	return service.postsSubscriptionInfo
}

func (service *service) RegisterSubscriber() uint32 {
	return atomic.AddUint32(&service.numberOfSubscribers, 1)
}

func (service *service) UnregisterSubscriber() uint32 {
	return atomic.AddUint32(&service.numberOfSubscribers, ^uint32(0))
}

func (service *service) GetGlobalProps() *proto.Props { return service.props }

func (service *service) GetProps(
	context.Context, *proto.PropsRequest) (*proto.Props, error) {
	return service.props, nil
}

func (service *service) Auth(
	ctx context.Context,
	request *proto.AuthRequest) (*proto.AuthResponse, error) {

	client, err := service.newRequestClient(ctx, "Auth")
	if err != nil {
		return nil, err
	}
	var token *string
	token, err = client.Auth(request)
	if err != nil {
		return nil, err
	}

	if token != nil {
		err := service.grpc.SendHeader(ctx, metadata.Pairs("auth", *token))
		if err != nil {
			return nil, client.RegisterError(codes.Internal,
				`failed to set auth-metadata: "%s"`, err)
		}
	}

	return &proto.AuthResponse{
			IsSuccess:      token != nil,
			AllowedMethods: client.GetAllowedMethods()},
		nil
}

func (service *service) ReadLog(
	request *proto.LogReadRequest, subscription proto.Au10_ReadLogServer) error {
	client, err := service.newRequestClient(subscription.Context(), "ReadLog")
	if err != nil {
		return err
	}
	return client.ReadLog(request, subscription)
}

func (service *service) ReadPosts(
	request *proto.PostsReadRequest,
	subscription proto.Au10_ReadPostsServer) error {
	client, err := service.newRequestClient(
		subscription.Context(), "ReadPosts")
	if err != nil {
		return err
	}
	return client.ReadPosts(request, subscription)
}

func (service *service) ReadMessage(
	request *proto.MessageReadRequest,
	subscription proto.Au10_ReadMessageServer) error {
	client, err := service.newRequestClient(
		subscription.Context(), "ReadMessage")
	if err != nil {
		return err
	}
	return client.ReadMessage(request, subscription)
}

func (service *service) AddVocal(
	ctx context.Context, request *proto.VocalAddRequest) (*proto.Vocal, error) {
	client, err := service.newRequestClient(ctx, "AddVocal")
	if err != nil {
		return nil, err
	}
	return client.AddVocal(request)
}

func (service *service) WriteMessageChunk(
	ctx context.Context,
	request *proto.MessageChunkWriteRequest) (*proto.MessageChunkWriteResponse, error) {
	client, err := service.newRequestClient(ctx, "WriteMessageChunk")
	if err != nil {
		return nil, err
	}
	return client.WriteMessageChunk(request)
}

func (service *service) newRequestClient(
	ctx context.Context, request string) (Client, error) {
	user := service.defaultUser
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if token, ok := md["auth"]; ok && len(token) == 1 {
			var err error
			user, err = service.GetUsers().FindSession(token[0])
			if err != nil {
				service.Log().Error(`Failed to find user session by token: "%s".`,
					err)
				return nil, status.Errorf(codes.Internal, "server internal error")
			}
			if user == nil {
				user = service.defaultUser
			}
		}
	}
	return service.newClient(
			atomic.AddUint32(&service.lastRequestID, 1), request, user, service),
		nil
}
