package accesspoint_test

import (
	"errors"
	"reflect"
	"testing"

	ap "bitbucket.org/au10/service/accesspoint/lib"
	proto "bitbucket.org/au10/service/accesspoint/proto"
	"bitbucket.org/au10/service/au10"
	mock_ap "bitbucket.org/au10/service/mock/accesspoint/lib"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	mock_context "bitbucket.org/au10/service/mock/context"
	mock_proto "bitbucket.org/au10/service/mock/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func Test_Accesspoint_Service_Methods(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	grpc := mock_ap.NewMockGrpc(mock)
	defaultUser := mock_au10.NewMockUser(mock)
	createClient := func(uint64, string, au10.User, ap.Service) ap.Client {
		assert.True(false)
		return nil
	}
	props := &proto.Props{}
	service := mock_au10.NewMockService(mock)
	accesspoint := ap.CreateAu10Server(
		props, defaultUser, createClient, grpc, service)

	ctx := mock_context.NewMockContext(mock)
	response, err := accesspoint.GetProps(ctx, nil)
	assert.True(response == props)
	assert.NoError(err)

	response = accesspoint.(ap.Service).GetGlobalProps()
	assert.True(response == props)
	assert.NoError(err)

	assert.True(accesspoint.(ap.Service).GetAu10() == service)

	logSubscriptionInfo := accesspoint.(ap.Service).GetLogSubscriptionInfo()
	assert.Equal(uint32(0), logSubscriptionInfo.NumberOfSubscribers)
	assert.Equal(1, reflect.Indirect(
		reflect.ValueOf(logSubscriptionInfo)).NumField())

	postsSubscriptionInfo := accesspoint.(ap.Service).GetPostsSubscriptionInfo()
	assert.Equal(uint32(0), postsSubscriptionInfo.NumberOfSubscribers)
	assert.Equal(1, reflect.Indirect(
		reflect.ValueOf(postsSubscriptionInfo)).NumField())

	assert.Equal(uint32(1), accesspoint.(ap.Service).RegisterSubscriber())
	assert.Equal(uint32(2), accesspoint.(ap.Service).RegisterSubscriber())
	assert.Equal(uint32(1), accesspoint.(ap.Service).UnregisterSubscriber())
	assert.Equal(uint32(0), accesspoint.(ap.Service).UnregisterSubscriber())
	assert.Equal(uint32(1), accesspoint.(ap.Service).RegisterSubscriber())
}

func Test_Accesspoint_Service_Auth(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	grpc := mock_ap.NewMockGrpc(mock)
	client := mock_ap.NewMockClient(mock)
	nextRequestID := uint64(1)
	defaultUser := mock_au10.NewMockUser(mock)
	expectedUser := &defaultUser
	createClient := func(
		requestID uint64,
		request string,
		user au10.User,
		service ap.Service) ap.Client {

		assert.Equal(nextRequestID, requestID)
		assert.Equal("Auth", request)
		nextRequestID++
		assert.True(user == *expectedUser)
		assert.NotNil(service)
		return client
	}
	service := mock_au10.NewMockService(mock)
	accesspoint := ap.CreateAu10Server(
		&proto.Props{}, defaultUser, createClient, grpc, service)
	ctx := mock_context.NewMockContext(mock)

	// No metadata
	authRequest := &proto.AuthRequest{}
	authExpectedResponse := "test auth response"
	client.EXPECT().Auth(authRequest).Return(&authExpectedResponse, nil)
	allowedMethods := &proto.AuthResponse_AllowedMethods{}
	client.EXPECT().GetAllowedMethods().Return(allowedMethods)
	grpc.EXPECT().
		SendHeader(ctx, metadata.MD{"auth": []string{authExpectedResponse}}).
		Return(nil)
	ctx.EXPECT().Value(gomock.Any()).Return(nil)
	response, err := accesspoint.Auth(ctx, authRequest)
	assert.NotNil(response)
	responseReflection := reflect.Indirect(reflect.ValueOf(response))
	assert.Equal(5, responseReflection.NumField())
	assert.True(response.IsSuccess)
	assert.True(allowedMethods == response.AllowedMethods)
	assert.NoError(err)

	// Metadata without auth
	authExpectedResponse += "!"
	client.EXPECT().Auth(authRequest).Return(&authExpectedResponse, nil)
	client.EXPECT().GetAllowedMethods().Return(allowedMethods)
	grpc.EXPECT().
		SendHeader(ctx, metadata.MD{"auth": []string{authExpectedResponse}}).
		Return(nil)
	ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"wrong": []string{authExpectedResponse}})
	response, err = accesspoint.Auth(ctx, authRequest)
	assert.NotNil(response)
	assert.Equal(5, responseReflection.NumField())
	assert.True(response.IsSuccess)
	assert.True(allowedMethods == response.AllowedMethods)
	assert.NoError(err)

	// Metadata with auth, but error at session finding
	authExpectedResponse += "!"
	ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	users := mock_au10.NewMockUsers(mock)
	users.EXPECT().FindSession(authExpectedResponse).
		Return(nil, errors.New("session finding error"))
	service.EXPECT().GetUsers().Return(users)
	log := mock_au10.NewMockLog(mock)
	log.EXPECT().Error(`Failed to find user session by token: "%s".`,
		errors.New("session finding error"))
	service.EXPECT().Log().Return(log)
	response, err = accesspoint.Auth(ctx, authRequest)
	assert.Nil(response)
	assert.EqualError(err,
		"rpc error: code = Internal desc = server internal error")

	// Metadata with auth, but filed to find session
	authExpectedResponse += "!"
	client.EXPECT().Auth(authRequest).Return(&authExpectedResponse, nil)
	client.EXPECT().GetAllowedMethods().Return(allowedMethods)
	grpc.EXPECT().
		SendHeader(ctx, metadata.MD{"auth": []string{authExpectedResponse}}).
		Return(nil)
	ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	users.EXPECT().FindSession(authExpectedResponse).Return(nil, nil)
	service.EXPECT().GetUsers().Return(users)
	response, err = accesspoint.Auth(ctx, authRequest)
	assert.NotNil(response)
	assert.Equal(5, responseReflection.NumField())
	assert.True(response.IsSuccess)
	assert.True(allowedMethods == response.AllowedMethods)
	assert.NoError(err)

	// Already authed
	authExpectedResponse += "!"
	client.EXPECT().Auth(authRequest).Return(&authExpectedResponse, nil)
	client.EXPECT().GetAllowedMethods().Return(allowedMethods)
	grpc.EXPECT().
		SendHeader(ctx, metadata.MD{"auth": []string{authExpectedResponse}}).
		Return(nil)
	ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	realUser := mock_au10.NewMockUser(mock)
	expectedUser = &realUser
	users.EXPECT().FindSession(authExpectedResponse).Return(realUser, nil)
	service.EXPECT().GetUsers().Return(users)
	response, err = accesspoint.Auth(ctx, authRequest)
	assert.NotNil(response)
	assert.Equal(5, responseReflection.NumField())
	assert.True(response.IsSuccess)
	assert.True(allowedMethods == response.AllowedMethods)
	assert.NoError(err)

	// Auth error
	authExpectedResponse += "!"
	client.EXPECT().Auth(authRequest).
		Return(&authExpectedResponse, errors.New("auth test error"))
	ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	expectedUser = &defaultUser
	users.EXPECT().FindSession(authExpectedResponse).Return(nil, nil)
	service.EXPECT().GetUsers().Return(users)
	response, err = accesspoint.Auth(ctx, authRequest)
	assert.Nil(response)
	assert.EqualError(err, "auth test error")

	// Wrong creds
	authExpectedResponse += "!"
	client.EXPECT().Auth(authRequest).Return(nil, nil)
	client.EXPECT().GetAllowedMethods().Return(allowedMethods)
	ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	users.EXPECT().FindSession(authExpectedResponse).Return(nil, nil)
	service.EXPECT().GetUsers().Return(users)
	response, err = accesspoint.Auth(ctx, authRequest)
	assert.NotNil(response)
	assert.Equal(5, responseReflection.NumField())
	assert.False(response.IsSuccess)
	assert.True(allowedMethods == response.AllowedMethods)
	assert.NoError(err)

	// Failed to store auth token
	authExpectedResponse += "!"
	client.EXPECT().Auth(authRequest).Return(&authExpectedResponse, nil)
	client.EXPECT().CreateError(codes.Internal,
		`failed to set auth-metadata: "%s"`, errors.New("test metadata error")).
		Return(errors.New("test auth error"))
	grpc.EXPECT().
		SendHeader(ctx, metadata.MD{"auth": []string{authExpectedResponse}}).
		Return(errors.New("test metadata error"))
	ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	users.EXPECT().FindSession(authExpectedResponse).Return(nil, nil)
	service.EXPECT().GetUsers().Return(users)
	response, err = accesspoint.Auth(ctx, authRequest)
	assert.Nil(response)
	assert.EqualError(err, "test auth error")
}

type serviceMethodTest struct {
	mock        *gomock.Controller
	assert      *assert.Assertions
	service     *mock_au10.MockService
	grpc        *mock_ap.MockGrpc
	client      *mock_ap.MockClient
	users       *mock_au10.MockUsers
	ctx         *mock_context.MockContext
	log         *mock_au10.MockLog
	accesspoint proto.Au10Server
}

func createTestServiceMethodTest(
	test *testing.T, request string) *serviceMethodTest {

	result := &serviceMethodTest{
		mock:   gomock.NewController(test),
		assert: assert.New(test)}
	result.grpc = mock_ap.NewMockGrpc(result.mock)
	result.client = mock_ap.NewMockClient(result.mock)
	result.service = mock_au10.NewMockService(result.mock)
	result.users = mock_au10.NewMockUsers(result.mock)
	result.ctx = mock_context.NewMockContext(result.mock)
	result.log = mock_au10.NewMockLog(result.mock)
	result.accesspoint = ap.CreateAu10Server(
		&proto.Props{},
		mock_au10.NewMockUser(result.mock),
		func(
			requestID uint64,
			actualRequest string,
			user au10.User,
			service ap.Service) ap.Client {
			result.assert.Equal(request, actualRequest)
			return result.client
		},
		result.grpc,
		result.service)
	return result
}

func (test *serviceMethodTest) close() {
	test.mock.Finish()
}

func (test *serviceMethodTest) testClientCreationError(
	runTest func() (interface{}, error)) {

	test.users.EXPECT().FindSession(gomock.Any()).
		Return(nil, errors.New("session finding error"))
	test.service.EXPECT().GetUsers().Return(test.users)
	test.log.EXPECT().Error(`Failed to find user session by token: "%s".`,
		errors.New("session finding error"))
	test.service.EXPECT().Log().Return(test.log)
	test.ctx.EXPECT().Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{""}})
	response, err := runTest()
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = Internal desc = server internal error")
}

func Test_Accesspoint_Service_ReadLog(t *testing.T) {
	test := createTestServiceMethodTest(t, "ReadLog")
	defer test.close()

	subscription := mock_proto.NewMockAu10_ReadLogServer(test.mock)

	test.testClientCreationError(func() (interface{}, error) {
		subscription.EXPECT().Context().Return(test.ctx)
		return nil, test.accesspoint.ReadLog(nil, subscription)
	})

	request := &proto.LogReadRequest{}
	subscription.EXPECT().Context().Return(test.ctx)
	test.ctx.EXPECT().Value(gomock.Any()).Return(nil)
	test.client.EXPECT().ReadLog(request, subscription).
		Return(errors.New("test error"))
	err := test.accesspoint.ReadLog(request, subscription)
	test.assert.EqualError(err, "test error")
}

func Test_Accesspoint_Service_ReadPosts(t *testing.T) {
	test := createTestServiceMethodTest(t, "ReadPosts")
	defer test.close()

	subscription := mock_proto.NewMockAu10_ReadPostsServer(test.mock)

	test.testClientCreationError(func() (interface{}, error) {
		subscription.EXPECT().Context().Return(test.ctx)
		return nil, test.accesspoint.ReadPosts(nil, subscription)
	})

	request := &proto.PostsReadRequest{}
	subscription.EXPECT().Context().Return(test.ctx)
	test.ctx.EXPECT().Value(gomock.Any()).Return(nil)
	test.client.EXPECT().ReadPosts(request, subscription).
		Return(errors.New("test error"))
	err := test.accesspoint.ReadPosts(request, subscription)
	test.assert.EqualError(err, "test error")
}

func Test_Accesspoint_Service_ReadMessage(t *testing.T) {
	test := createTestServiceMethodTest(t, "ReadMessage")
	defer test.close()

	subscription := mock_proto.NewMockAu10_ReadMessageServer(test.mock)

	test.testClientCreationError(func() (interface{}, error) {
		subscription.EXPECT().Context().Return(test.ctx)
		return nil, test.accesspoint.ReadMessage(nil, subscription)
	})

	request := &proto.MessageReadRequest{}
	subscription.EXPECT().Context().Return(test.ctx)
	test.ctx.EXPECT().Value(gomock.Any()).Return(nil)
	test.client.EXPECT().ReadMessage(request, subscription).
		Return(errors.New("test error"))
	err := test.accesspoint.ReadMessage(request, subscription)
	test.assert.EqualError(err, "test error")
}

func Test_Accesspoint_Service_AddVocal(t *testing.T) {
	test := createTestServiceMethodTest(t, "AddVocal")
	defer test.close()

	test.testClientCreationError(func() (interface{}, error) {
		return test.accesspoint.AddVocal(test.ctx, nil)
	})

	request := &proto.VocalAddRequest{}
	expectedResponse := &proto.Vocal{}
	test.ctx.EXPECT().Value(gomock.Any()).Return(nil)
	test.client.EXPECT().AddVocal(request).
		Return(expectedResponse, errors.New("test error"))
	response, err := test.accesspoint.AddVocal(test.ctx, request)
	test.assert.True(response == expectedResponse)
	test.assert.EqualError(err, "test error")
}

func Test_Accesspoint_Service_MessageChunkWrite(t *testing.T) {
	test := createTestServiceMethodTest(t, "WriteMessageChunk")
	defer test.close()

	test.testClientCreationError(func() (interface{}, error) {
		return test.accesspoint.WriteMessageChunk(test.ctx, nil)
	})

	request := &proto.MessageChunkWriteRequest{}
	expectedResponse := &proto.MessageChunkWriteResponse{}
	test.ctx.EXPECT().Value(gomock.Any()).Return(nil)
	test.client.EXPECT().WriteMessageChunk(request).
		Return(expectedResponse, errors.New("test error"))
	response, err := test.accesspoint.WriteMessageChunk(test.ctx, request)
	test.assert.True(response == expectedResponse)
	test.assert.EqualError(err, "test error")
}
