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

type serviceTest struct {
	mock   *gomock.Controller
	assert *assert.Assertions

	props     *proto.Props
	grpc      *mock_ap.MockGrpc
	client    *mock_ap.MockClient
	posts     *mock_au10.MockPosts
	publisher *mock_au10.MockPublisher
	users     *mock_au10.MockUsers
	user      *mock_au10.MockUser
	ctx       *mock_context.MockContext
	log       *mock_au10.MockLog
	logReader *mock_au10.MockLogReader

	service proto.Au10Server
}

func newServiceTestObj(test *testing.T) *serviceTest {
	result := &serviceTest{
		mock:   gomock.NewController(test),
		assert: assert.New(test),
		props:  &proto.Props{}}
	result.grpc = mock_ap.NewMockGrpc(result.mock)
	result.client = mock_ap.NewMockClient(result.mock)
	result.posts = mock_au10.NewMockPosts(result.mock)
	result.publisher = mock_au10.NewMockPublisher(result.mock)
	result.users = mock_au10.NewMockUsers(result.mock)
	result.user = mock_au10.NewMockUser(result.mock)
	result.ctx = mock_context.NewMockContext(result.mock)
	result.log = mock_au10.NewMockLog(result.mock)
	return result
}

func (test *serviceTest) init(
	newClient func(uint32, string, au10.User, ap.Service) ap.Client) {
	test.service = ap.NewAu10Server(test.props, test.log, test.logReader,
		test.posts, test.publisher, test.users, test.user, newClient, test.grpc)
}

func newServiceMethodTest(test *testing.T, request string) *serviceTest {
	result := newServiceTestObj(test)
	result.init(
		func(
			requestID uint32,
			actualRequest string,
			user au10.User,
			service ap.Service) ap.Client {
			result.assert.Equal(request, actualRequest)
			return result.client
		})
	return result
}

func (test *serviceTest) close() {
	test.mock.Finish()
}

func Test_Accesspoint_Service_Methods(t *testing.T) {
	test := newServiceTestObj(t)
	defer test.close()
	test.init(func(uint32, string, au10.User, ap.Service) ap.Client {
		test.assert.True(false)
		return nil
	})

	props, err := test.service.GetProps(test.ctx, nil)
	test.assert.True(props == test.props)
	test.assert.NoError(err)

	props = test.service.(ap.Service).GetGlobalProps()
	test.assert.True(props == test.props)
	test.assert.NoError(err)

	logSubscriptionInfo := test.service.(ap.Service).GetLogSubscriptionInfo()
	test.assert.Equal(uint32(0), logSubscriptionInfo.NumberOfSubscribers)
	test.assert.Equal(1, reflect.Indirect(
		reflect.ValueOf(logSubscriptionInfo)).NumField())

	postsSubscriptionInfo := test.service.(ap.Service).GetPostsSubscriptionInfo()
	test.assert.Equal(uint32(0), postsSubscriptionInfo.NumberOfSubscribers)
	test.assert.Equal(1, reflect.Indirect(
		reflect.ValueOf(postsSubscriptionInfo)).NumField())

	test.assert.True(test.service.(ap.Service).Log() == test.log)
	test.assert.True(test.service.(ap.Service).GetLogReader() == test.logReader)
	test.assert.True(test.service.(ap.Service).GetPosts() == test.posts)
	test.assert.True(test.service.(ap.Service).GetPublisher() == test.publisher)
	test.assert.True(test.service.(ap.Service).GetUsers() == test.users)
	test.assert.NotNil(test.service.(ap.Service).Convert())
	test.assert.Equal(uint32(1), test.service.(ap.Service).RegisterSubscriber())
	test.assert.Equal(uint32(2), test.service.(ap.Service).RegisterSubscriber())
	test.assert.Equal(uint32(1), test.service.(ap.Service).UnregisterSubscriber())
	test.assert.Equal(uint32(0), test.service.(ap.Service).UnregisterSubscriber())
	test.assert.Equal(uint32(1), test.service.(ap.Service).RegisterSubscriber())
}

func Test_Accesspoint_Service_Auth(t *testing.T) {
	test := newServiceTestObj(t)
	defer test.close()
	nextRequestID := uint32(1)
	expectedUser := &test.user
	test.init(func(
		requestID uint32,
		request string,
		user au10.User,
		service ap.Service) ap.Client {
		test.assert.Equal(nextRequestID, requestID)
		test.assert.Equal("Auth", request)
		nextRequestID++
		test.assert.True(user == *expectedUser)
		test.assert.NotNil(service)
		return test.client
	})

	// No metadata
	authRequest := &proto.AuthRequest{}
	authExpectedResponse := "test auth response"
	test.client.EXPECT().Auth(authRequest).Return(&authExpectedResponse, nil)
	allowedMethods := &proto.AuthResponse_AllowedMethods{}
	test.client.EXPECT().GetAllowedMethods().Return(allowedMethods)
	test.grpc.EXPECT().
		SendHeader(test.ctx, metadata.MD{"auth": []string{authExpectedResponse}}).
		Return(nil)
	test.ctx.EXPECT().Value(gomock.Any()).Return(nil)
	response, err := test.service.Auth(test.ctx, authRequest)
	test.assert.NotNil(response)
	responseReflection := reflect.Indirect(reflect.ValueOf(response))
	test.assert.Equal(5, responseReflection.NumField())
	test.assert.True(response.IsSuccess)
	test.assert.True(allowedMethods == response.AllowedMethods)
	test.assert.NoError(err)

	// Metadata without auth
	authExpectedResponse += "!"
	test.client.EXPECT().Auth(authRequest).Return(&authExpectedResponse, nil)
	test.client.EXPECT().GetAllowedMethods().Return(allowedMethods)
	test.grpc.EXPECT().
		SendHeader(test.ctx, metadata.MD{"auth": []string{authExpectedResponse}}).
		Return(nil)
	test.ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"wrong": []string{authExpectedResponse}})
	response, err = test.service.Auth(test.ctx, authRequest)
	test.assert.NotNil(response)
	test.assert.Equal(5, responseReflection.NumField())
	test.assert.True(response.IsSuccess)
	test.assert.True(allowedMethods == response.AllowedMethods)
	test.assert.NoError(err)

	// Metadata with auth, but error at session finding
	authExpectedResponse += "!"
	test.ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	test.users.EXPECT().FindSession(authExpectedResponse).
		Return(nil, errors.New("session finding error"))
	test.log.EXPECT().Error(`Failed to find user session by token: "%s".`,
		errors.New("session finding error"))
	response, err = test.service.Auth(test.ctx, authRequest)
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = Internal desc = server internal error")

	// Metadata with auth, but filed to find session
	authExpectedResponse += "!"
	test.client.EXPECT().Auth(authRequest).Return(&authExpectedResponse, nil)
	test.client.EXPECT().GetAllowedMethods().Return(allowedMethods)
	test.grpc.EXPECT().
		SendHeader(test.ctx, metadata.MD{"auth": []string{authExpectedResponse}}).
		Return(nil)
	test.ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	test.users.EXPECT().FindSession(authExpectedResponse).Return(nil, nil)
	response, err = test.service.Auth(test.ctx, authRequest)
	test.assert.NotNil(response)
	test.assert.Equal(5, responseReflection.NumField())
	test.assert.True(response.IsSuccess)
	test.assert.True(allowedMethods == response.AllowedMethods)
	test.assert.NoError(err)

	// Already authed
	authExpectedResponse += "!"
	test.client.EXPECT().Auth(authRequest).Return(&authExpectedResponse, nil)
	test.client.EXPECT().GetAllowedMethods().Return(allowedMethods)
	test.grpc.EXPECT().
		SendHeader(test.ctx, metadata.MD{"auth": []string{authExpectedResponse}}).
		Return(nil)
	test.ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	realUser := mock_au10.NewMockUser(test.mock)
	expectedUser = &realUser
	test.users.EXPECT().FindSession(authExpectedResponse).Return(realUser, nil)
	response, err = test.service.Auth(test.ctx, authRequest)
	test.assert.NotNil(response)
	test.assert.Equal(5, responseReflection.NumField())
	test.assert.True(response.IsSuccess)
	test.assert.True(allowedMethods == response.AllowedMethods)
	test.assert.NoError(err)

	// Auth error
	authExpectedResponse += "!"
	test.client.EXPECT().Auth(authRequest).
		Return(&authExpectedResponse, errors.New("auth test error"))
	test.ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	expectedUser = &test.user
	test.users.EXPECT().FindSession(authExpectedResponse).Return(nil, nil)
	response, err = test.service.Auth(test.ctx, authRequest)
	test.assert.Nil(response)
	test.assert.EqualError(err, "auth test error")

	// Wrong creds
	authExpectedResponse += "!"
	test.client.EXPECT().Auth(authRequest).Return(nil, nil)
	test.client.EXPECT().GetAllowedMethods().Return(allowedMethods)
	test.ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	test.users.EXPECT().FindSession(authExpectedResponse).Return(nil, nil)
	response, err = test.service.Auth(test.ctx, authRequest)
	test.assert.NotNil(response)
	test.assert.Equal(5, responseReflection.NumField())
	test.assert.False(response.IsSuccess)
	test.assert.True(allowedMethods == response.AllowedMethods)
	test.assert.NoError(err)

	// Failed to store auth token
	authExpectedResponse += "!"
	test.client.EXPECT().Auth(authRequest).Return(&authExpectedResponse, nil)
	test.client.EXPECT().RegisterError(codes.Internal,
		`failed to set auth-metadata: "%s"`, errors.New("test metadata error")).
		Return(errors.New("test auth error"))
	test.grpc.EXPECT().
		SendHeader(test.ctx, metadata.MD{"auth": []string{authExpectedResponse}}).
		Return(errors.New("test metadata error"))
	test.ctx.EXPECT().
		Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{authExpectedResponse}})
	test.users.EXPECT().FindSession(authExpectedResponse).Return(nil, nil)
	response, err = test.service.Auth(test.ctx, authRequest)
	test.assert.Nil(response)
	test.assert.EqualError(err, "test auth error")
}

func (test *serviceTest) testClientCreationError(
	runTest func() (interface{}, error)) {
	test.users.EXPECT().FindSession(gomock.Any()).
		Return(nil, errors.New("session finding error"))
	test.log.EXPECT().Error(`Failed to find user session by token: "%s".`,
		errors.New("session finding error"))
	test.ctx.EXPECT().Value(gomock.Any()).
		Return(metadata.MD{"auth": []string{""}})
	response, err := runTest()
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = Internal desc = server internal error")
}

func Test_Accesspoint_Service_ReadLog(t *testing.T) {
	test := newServiceMethodTest(t, "ReadLog")
	defer test.close()

	subscription := mock_proto.NewMockAu10_ReadLogServer(test.mock)

	test.testClientCreationError(func() (interface{}, error) {
		subscription.EXPECT().Context().Return(test.ctx)
		return nil, test.service.ReadLog(nil, subscription)
	})

	request := &proto.LogReadRequest{}
	subscription.EXPECT().Context().Return(test.ctx)
	test.ctx.EXPECT().Value(gomock.Any()).Return(nil)
	test.client.EXPECT().ReadLog(request, subscription).
		Return(errors.New("test error"))
	err := test.service.ReadLog(request, subscription)
	test.assert.EqualError(err, "test error")
}

func Test_Accesspoint_Service_ReadPosts(t *testing.T) {
	test := newServiceMethodTest(t, "ReadPosts")
	defer test.close()

	subscription := mock_proto.NewMockAu10_ReadPostsServer(test.mock)

	test.testClientCreationError(func() (interface{}, error) {
		subscription.EXPECT().Context().Return(test.ctx)
		return nil, test.service.ReadPosts(nil, subscription)
	})

	request := &proto.PostsReadRequest{}
	subscription.EXPECT().Context().Return(test.ctx)
	test.ctx.EXPECT().Value(gomock.Any()).Return(nil)
	test.client.EXPECT().ReadPosts(request, subscription).
		Return(errors.New("test error"))
	err := test.service.ReadPosts(request, subscription)
	test.assert.EqualError(err, "test error")
}

func Test_Accesspoint_Service_ReadMessage(t *testing.T) {
	test := newServiceMethodTest(t, "ReadMessage")
	defer test.close()

	subscription := mock_proto.NewMockAu10_ReadMessageServer(test.mock)

	test.testClientCreationError(func() (interface{}, error) {
		subscription.EXPECT().Context().Return(test.ctx)
		return nil, test.service.ReadMessage(nil, subscription)
	})

	request := &proto.MessageReadRequest{}
	subscription.EXPECT().Context().Return(test.ctx)
	test.ctx.EXPECT().Value(gomock.Any()).Return(nil)
	test.client.EXPECT().ReadMessage(request, subscription).
		Return(errors.New("test error"))
	err := test.service.ReadMessage(request, subscription)
	test.assert.EqualError(err, "test error")
}

func Test_Accesspoint_Service_AddVocal(t *testing.T) {
	test := newServiceMethodTest(t, "AddVocal")
	defer test.close()

	test.testClientCreationError(func() (interface{}, error) {
		return test.service.AddVocal(test.ctx, nil)
	})

	request := &proto.VocalAddRequest{}
	expectedResponse := &proto.Vocal{}
	test.ctx.EXPECT().Value(gomock.Any()).Return(nil)
	test.client.EXPECT().AddVocal(request).
		Return(expectedResponse, errors.New("test error"))
	response, err := test.service.AddVocal(test.ctx, request)
	test.assert.True(response == expectedResponse)
	test.assert.EqualError(err, "test error")
}

func Test_Accesspoint_Service_MessageChunkWrite(t *testing.T) {
	test := newServiceMethodTest(t, "WriteMessageChunk")
	defer test.close()

	test.testClientCreationError(func() (interface{}, error) {
		return test.service.WriteMessageChunk(test.ctx, nil)
	})

	request := &proto.MessageChunkWriteRequest{}
	expectedResponse := &proto.MessageChunkWriteResponse{}
	test.ctx.EXPECT().Value(gomock.Any()).Return(nil)
	test.client.EXPECT().WriteMessageChunk(request).
		Return(expectedResponse, errors.New("test error"))
	response, err := test.service.WriteMessageChunk(test.ctx, request)
	test.assert.True(response == expectedResponse)
	test.assert.EqualError(err, "test error")
}
