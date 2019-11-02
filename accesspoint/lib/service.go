package accesspoint

import (
	"context"

	"bitbucket.org/au10/service/au10"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type service struct {
	au10        au10.Service
	defaultUser au10.User

	numberOfSubscribers uint32
	logSubscriptionInfo subscriptionInfo
}

// CreateService creates new AccessPointHandler instance.
func CreateService(
	defaultUser au10.User,
	au10Service au10.Service) AccessPointServer {

	return &service{
		au10:                au10Service,
		defaultUser:         defaultUser,
		logSubscriptionInfo: subscriptionInfo{name: "log"}}
}

func (service *service) Auth(
	ctx context.Context,
	request *AuthRequest) (*AuthResponse, error) {

	client, err := service.createClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Auth(request)
}

func (service *service) ReadLog(
	request *LogReadRequest, subscription AccessPoint_ReadLogServer) error {

	client, err := service.createClient(subscription.Context())
	if err != nil {
		return err
	}
	return client.SubscribeToLog(request, subscription)
}

func (service *service) createClient(ctx context.Context) (Client, error) {
	var user au10.User
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if token, ok := md["auth"]; ok && len(token) == 1 {
			var err error
			user, err := service.au10.GetUsers().FindSession(token[0])
			if err != nil {
				service.au10.Log().Error(
					`Failed to find user session by token: "%s".`, err)
				return nil, status.Errorf(codes.Internal, "server internal error")
			}
			if user == nil {
				user = service.defaultUser
			}
		} else {
			user = service.defaultUser
		}
	}
	return CreateClient(user, service), nil
}
