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
	GetAu10() au10.Service
	GetGlobalProps() *proto.Props

	GetLogSubscriptionInfo() *SubscriptionInfo
	RegisterSubscriber() uint32
	UnregisterSubscriber() uint32
}

type service struct {
	props *proto.Props
	au10  au10.Service

	defaultUser  au10.User
	createClient func(requestID uint64, user au10.User, service Service) Client
	grpc         Grpc

	lastRequestID       uint64
	numberOfSubscribers uint32
	logSubscriptionInfo *SubscriptionInfo
}

// CreateAu10Server creates Au10Server server implementation instance.
func CreateAu10Server(
	props *proto.Props,
	defaultUser au10.User,
	createClient func(requestID uint64, user au10.User, service Service) Client,
	grpc Grpc,
	au10Service au10.Service) proto.Au10Server {

	return &service{
		props:               props,
		au10:                au10Service,
		defaultUser:         defaultUser,
		createClient:        createClient,
		grpc:                grpc,
		logSubscriptionInfo: &SubscriptionInfo{Name: "log"}}
}

func (service *service) GetAu10() au10.Service { return service.au10 }

func (service *service) GetLogSubscriptionInfo() *SubscriptionInfo {
	return service.logSubscriptionInfo
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

	client, err := service.createRequestClient(ctx)
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
			return nil, client.CreateError(codes.Internal,
				`failed to set auth-metadata: "%s"`, err)
		}
	}

	return &proto.AuthResponse{
			IsSuccess: token != nil,
			Methods:   client.GetAvailableMethods()},
		nil
}

func (service *service) ReadLog(
	request *proto.LogReadRequest, subscription proto.Au10_ReadLogServer) error {

	client, err := service.createRequestClient(subscription.Context())
	if err != nil {
		return err
	}
	return client.ReadLog(request, subscription)
}

func (service *service) ReadPosts(
	request *proto.PostsReadRequest,
	subscription proto.Au10_ReadPostsServer) error {

	client, err := service.createRequestClient(subscription.Context())
	if err != nil {
		return err
	}
	return client.ReadPosts(request, subscription)
}

func (service *service) ReadMessage(
	request *proto.MessageReadRequest,
	subscription proto.Au10_ReadMessageServer) error {

	client, err := service.createRequestClient(subscription.Context())
	if err != nil {
		return err
	}
	return client.ReadMessage(request, subscription)
}

func (service *service) AddPost(
	ctx context.Context,
	request *proto.PostAddRequest) (*proto.PostAddResponse, error) {

	client, err := service.createRequestClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.AddPost(request)
}

func (service *service) MessageChunkWrite(
	ctx context.Context,
	request *proto.MessageChunkWriteRequest) (*proto.MessageChunkWriteResponse, error) {

	client, err := service.createRequestClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.MessageChunkWrite(request)
}

func (service *service) createRequestClient(
	ctx context.Context) (Client, error) {

	user := service.defaultUser
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if token, ok := md["auth"]; ok && len(token) == 1 {
			var err error
			user, err = service.au10.GetUsers().FindSession(token[0])
			if err != nil {
				service.au10.Log().Error(`Failed to find user session by token: "%s".`,
					err)
				return nil, status.Errorf(codes.Internal, "server internal error")
			}
			if user == nil {
				user = service.defaultUser
			}
		}
	}
	return service.createClient(
			atomic.AddUint64(&service.lastRequestID, 1), user, service),
		nil
}
