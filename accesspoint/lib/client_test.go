package accesspoint_test

import (
	"context"
	"errors"
	fmt "fmt"
	"testing"

	ap "bitbucket.org/au10/service/accesspoint/lib"
	proto "bitbucket.org/au10/service/accesspoint/proto"
	au10 "bitbucket.org/au10/service/au10"
	mock_ap "bitbucket.org/au10/service/mock/accesspoint/lib"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	mock_context "bitbucket.org/au10/service/mock/context"
	mock_proto "bitbucket.org/au10/service/mock/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func Test_Accesspoint_Client_Log(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()

	service := mock_ap.NewMockService(mock)
	user := mock_au10.NewMockUser(mock)

	client := ap.NewClient(123, "Test", user, service)

	log := mock_au10.NewMockLog(mock)
	service.EXPECT().Log().Times(4).Return(log)

	user.EXPECT().GetID().Return(au10.UserID(1))
	log.EXPECT().Error("[Test/123/1] test error record %s", "argument")
	client.LogError("test error record %s", "argument")

	user.EXPECT().GetID().Return(au10.UserID(2))
	log.EXPECT().Warn("[Test/123/2] test warn record %s", "argument")
	client.LogWarn("test warn record %s", "argument")

	user.EXPECT().GetID().Return(au10.UserID(3))
	log.EXPECT().Info("[Test/123/3] test info record %s", "argument")
	client.LogInfo("test info record %s", "argument")

	user.EXPECT().GetID().Return(au10.UserID(4))
	log.EXPECT().Debug("[Test/123/4] test debug record %s", "argument")
	client.LogDebug("test debug record %s", "argument")
}

type clientTest struct {
	mock          *gomock.Controller
	assert        *assert.Assertions
	service       *mock_ap.MockService
	convert       *mock_ap.MockConvertor
	log           *mock_au10.MockLog
	user          *mock_au10.MockUser
	logMembership *mock_au10.MockMembership
	rights        []au10.Rights
	client        ap.Client
}

func newClientTest(test *testing.T) *clientTest {
	result := &clientTest{
		mock:   gomock.NewController(test),
		assert: assert.New(test),
		rights: []au10.Rights{
			au10.NewRights("x1", "z1"),
			au10.NewRights("x2", "z2")}}
	result.service = mock_ap.NewMockService(result.mock)
	result.convert = mock_ap.NewMockConvertor(result.mock)
	result.log = mock_au10.NewMockLog(result.mock)
	result.user = mock_au10.NewMockUser(result.mock)
	result.logMembership = mock_au10.NewMockMembership(result.mock)
	result.client = ap.NewClient(345, "Test", result.user, result.service)

	result.service.EXPECT().Log().AnyTimes().Return(result.log)
	result.service.EXPECT().Convert().AnyTimes().Return(result.convert)
	result.log.EXPECT().GetMembership().AnyTimes().Return(result.logMembership)
	result.user.EXPECT().GetID().AnyTimes().Return(au10.UserID(836))
	result.user.EXPECT().GetRights().AnyTimes().Return(result.rights)

	return result
}

func (test *clientTest) close() {
	test.mock.Finish()
}

func (test *clientTest) testPermissionDenied(
	membership *mock_au10.MockMembership,
	run func() (interface{}, error),
	action string) {

	membership.EXPECT().IsAllowed(test.rights).Return(false)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	test.log.EXPECT().Error(
		fmt.Sprintf(
			"[Test/345/836] Permission denied for %s (RPC error %%d).",
			action),
		codes.PermissionDenied)
	response, err := run()
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = PermissionDenied desc = PERMISSION_DENIED")
}

func (test *clientTest) testSubscribeError(
	membership *mock_au10.MockMembership,
	prepareErr func(error),
	run func() (interface{}, error),
	action string) {

	test.testPermissionDenied(membership, run, action)

	membership.EXPECT().IsAllowed(test.rights).Return(true)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	prepareErr(errors.New("test subscribe error"))
	test.log.EXPECT().Error(
		`[Test/345/836] Failed to subscribe: "test subscribe error" (RPC error %d).`,
		codes.Internal)
	response, err := run()
	test.assert.Nil(response)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")

}

func (test *clientTest) testSubscription(
	prepare func(subscriptionInfo *ap.SubscriptionInfo, isErrData bool),
	run func(ctx context.Context, isErrData bool) error,
	createData func(),
	createErrorData []struct {
		errText string
		run     func()
	},
	createErr func(error),
	closeSubscription func()) {

	// Successfull execution

	ctx := mock_context.NewMockContext(test.mock)
	ctxDoneChan := make(chan struct{})
	defer close(ctxDoneChan)
	ctx.EXPECT().Done().MinTimes(1).Return(ctxDoneChan)

	subscriptionInfo := &ap.SubscriptionInfo{NumberOfSubscribers: 101}
	testNumberOfStructFields(test.assert, subscriptionInfo, 1)

	test.log.EXPECT().Debug("[Test/345/836] Canceled (%d/%d).",
		uint32(101), uint32(909)).
		After(test.log.EXPECT().Debug("[Test/345/836] Subscribed (%d/%d).",
			uint32(102), uint32(808)))

	prepare(subscriptionInfo, false)

	test.service.EXPECT().UnregisterSubscriber().Return(uint32(909)).After(
		test.service.EXPECT().RegisterSubscriber().Return(uint32(808)))

	go func() {
		createData()
		ctxDoneChan <- struct{}{}
	}()
	test.assert.NoError(run(ctx, false))

	// Failed execution

	subscriptionInfo.NumberOfSubscribers = 201

	test.log.EXPECT().Error(
		`[Test/345/836] Failed: "test subscription error" (201/2909) (RPC error %d).`,
		codes.Internal).
		After(test.log.EXPECT().Debug("[Test/345/836] Subscribed (%d/%d).",
			uint32(202), uint32(2808)))

	prepare(subscriptionInfo, false)

	test.service.EXPECT().UnregisterSubscriber().Return(uint32(2909)).After(
		test.service.EXPECT().RegisterSubscriber().Return(uint32(2808)))

	go func() {
		createData()
		test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
		createErr(errors.New("test subscription error"))
	}()
	test.assert.EqualError(run(ctx, false),
		"rpc error: code = Internal desc = INTERNAL")

	// Closed

	subscriptionInfo.NumberOfSubscribers = 301

	test.log.EXPECT().Debug("[Test/345/836] Canceled (%d/%d).",
		uint32(301), uint32(3909)).
		After(test.log.EXPECT().Debug("[Test/345/836] Subscribed (%d/%d).",
			uint32(302), uint32(3808)))

	prepare(subscriptionInfo, false)

	test.service.EXPECT().UnregisterSubscriber().Return(uint32(3909)).After(
		test.service.EXPECT().RegisterSubscriber().Return(uint32(3808)))

	go func() {
		createData()
		closeSubscription()
	}()
	test.assert.NoError(run(ctx, false))

	// Data with error

	subscriptionInfo.NumberOfSubscribers = 401

	for _, dataMock := range createErrorData {

		test.log.EXPECT().Error(
			`[Test/345/836] Failed: "`+dataMock.errText+`" (401/4909) (RPC error %d).`,
			codes.Internal).
			After(test.log.EXPECT().Debug("[Test/345/836] Subscribed (%d/%d).",
				uint32(402), uint32(4808)))

		prepare(subscriptionInfo, true)

		test.service.EXPECT().UnregisterSubscriber().Return(uint32(4909)).After(
			test.service.EXPECT().RegisterSubscriber().Return(uint32(4808)))

		go func() {
			test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
			dataMock.run()
		}()
		test.assert.EqualError(run(ctx, true),
			"rpc error: code = Internal desc = INTERNAL")

	}
}

func Test_Accesspoint_Client_RegisterError(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	test.log.EXPECT().Error("[Test/345/836] Test error full text (RPC error %d).",
		codes.Internal)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	err := test.client.RegisterError(codes.Internal, "test error full text")
	test.assert.EqualError(err,
		"rpc error: code = Internal desc = INTERNAL")

	test.log.EXPECT().Error("[Test/345/836] RPC error %d.", codes.Code(18))
	test.log.EXPECT().Error(
		"[Test/345/836] Failed to find external description for status code %d.",
		codes.Code(18))
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	err = test.client.RegisterError(codes.Code(18), "")
	test.assert.EqualError(err, "rpc error: code = Code(18) desc = unknown error")

	test.user.EXPECT().BlockByProtocolMismatch()
	test.log.EXPECT().Error("[Test/345/836] Test error full text 2 (RPC error %d).",
		codes.InvalidArgument)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(true)
	err = test.client.RegisterError(codes.InvalidArgument, "test error full text 2")
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = test error full text 2")
}

func Test_Accesspoint_Client_Auth(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	users := mock_au10.NewMockUsers(test.mock)
	test.service.EXPECT().GetUsers().Times(3).Return(users)

	user := mock_au10.NewMockUser(test.mock)

	testToken := "test token"
	users.EXPECT().Auth("test login x").
		Return(user, &testToken, errors.New("test error"))
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	test.log.EXPECT().Error(
		`[Test/345/836] Failed to auth "test login x": "test error" (RPC error %d).`,
		codes.Internal)
	token, err := test.client.Auth(&proto.AuthRequest{Login: "test login x"})
	test.assert.Nil(token)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")

	users.EXPECT().Auth("test login x2").Return(user, nil, nil)
	test.log.EXPECT().Info(
		`[Test/345/836] Wrong creds for "%s"`, "test login x2")
	token, err = test.client.Auth(&proto.AuthRequest{Login: "test login x2"})
	test.assert.Nil(token)
	test.assert.NoError(err)

	users.EXPECT().Auth("test login x3").Return(user, &testToken, nil)
	user.EXPECT().GetID().Return(au10.UserID(1))
	test.log.EXPECT().Info(`[Test/345/1] Authed "%s" (was %d).`,
		"test login x3", au10.UserID(836))
	token, err = test.client.Auth(&proto.AuthRequest{Login: "test login x3"})
	test.assert.NotNil(token)
	test.assert.Equal("test token", *token)
	test.assert.NoError(err)
}

func Test_Accesspoint_Client_ReadLog(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	response := mock_proto.NewMockAu10_ReadLogServer(test.mock)

	membership := mock_au10.NewMockMembership(test.mock)

	logReader := mock_au10.NewMockLogReader(test.mock)
	logReader.EXPECT().GetMembership().MinTimes(1).Return(membership)
	test.service.EXPECT().GetLogReader().MinTimes(1).Return(logReader)

	test.testSubscribeError(
		membership,
		func(err error) { logReader.EXPECT().Subscribe().Return(nil, err) },
		func() (interface{}, error) {
			return nil, test.client.ReadLog(nil, response)
		},
		"log")

	sourceRecord := mock_au10.NewMockLogRecord(test.mock)
	protoRecord := &proto.LogRecord{SeqNum: 97531}

	var recordsChan chan au10.LogRecord
	var errChan chan error
	test.testSubscription(
		func(subscriptionInfo *ap.SubscriptionInfo, isErrData bool) {
			test.service.EXPECT().GetLogSubscriptionInfo().Return(subscriptionInfo)
			test.convert.EXPECT().LogRecordToProto(sourceRecord).Return(protoRecord)
			recordsChan = make(chan au10.LogRecord)
		},
		func(ctx context.Context, isErrData bool) error {
			test.assert.False(isErrData)
			subscription := mock_au10.NewMockLogSubscription(test.mock)
			errChan = make(chan error)
			subscription.EXPECT().GetErrChan().MinTimes(1).Return(errChan)
			subscription.EXPECT().GetRecordsChan().MinTimes(1).Return(recordsChan)
			subscription.EXPECT().Close().Do(func() {
				if recordsChan != nil {
					close(recordsChan)
				}
				close(errChan)
			})
			logReader.EXPECT().Subscribe().Return(subscription, nil)
			membership.EXPECT().IsAllowed(test.rights).Return(true)
			response.EXPECT().Send(protoRecord).Return(nil)
			response.EXPECT().Context().Return(ctx)
			return test.client.ReadLog(nil, response)
		},
		func() { recordsChan <- sourceRecord },
		nil,
		func(err error) { errChan <- err },
		func() {
			close(recordsChan)
			recordsChan = nil
		})
}

func Test_Accesspoint_Client_ReadPosts(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	postsMembership := mock_au10.NewMockMembership(test.mock)

	posts := mock_au10.NewMockPosts(test.mock)
	posts.EXPECT().GetMembership().MinTimes(1).Return(postsMembership)
	test.service.EXPECT().GetPosts().MinTimes(1).Return(posts)

	response := mock_proto.NewMockAu10_ReadPostsServer(test.mock)

	test.testSubscribeError(
		postsMembership,
		func(err error) { posts.EXPECT().Subscribe().Return(nil, err) },
		func() (interface{}, error) {
			return nil, test.client.ReadPosts(nil, response)
		},
		"posts")

	sourceRecord := mock_au10.NewMockVocal(test.mock)
	protoRecord := &proto.Vocal{Post: &proto.Post{Id: "8193"}}

	var postsChan chan au10.Post
	var errChan chan error
	test.testSubscription(
		func(subscriptionInfo *ap.SubscriptionInfo, isErrData bool) {
			test.service.EXPECT().GetPostsSubscriptionInfo().Return(subscriptionInfo)
			if !isErrData {
				test.convert.EXPECT().VocalToProto(sourceRecord, gomock.Any()).
					Return(protoRecord)
			}
			postsChan = make(chan au10.Post)
		},
		func(ctx context.Context, isErrData bool) error {
			subscription := mock_au10.NewMockPostsSubscription(test.mock)
			errChan = make(chan error)
			subscription.EXPECT().GetErrChan().MinTimes(1).Return(errChan)
			subscription.EXPECT().GetRecordsChan().MinTimes(1).Return(postsChan)
			subscription.EXPECT().Close().Do(func() {
				if postsChan != nil {
					close(postsChan)
				}
				close(errChan)
			})
			posts.EXPECT().Subscribe().Return(subscription, nil)
			postsMembership.EXPECT().IsAllowed(test.rights).Return(true)
			if !isErrData {
				response.EXPECT().Send(&proto.PostUpdate{
					Post: &proto.PostUpdate_Vocal{Vocal: protoRecord}}).
					Return(nil)
			}
			response.EXPECT().Context().Return(ctx)
			return test.client.ReadPosts(nil, response)
		},
		func() { postsChan <- sourceRecord },
		[]struct {
			errText string
			run     func()
		}{
			struct {
				errText string
				run     func()
			}{
				errText: `unknown post type "<nil>"`,
				run:     func() { postsChan <- nil }}},
		func(err error) { errChan <- err },
		func() {
			close(postsChan)
			postsChan = nil
		})
}

func Test_Accesspoint_Client_AddVocal_PermissionDenied(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	test.log.EXPECT().Error(
		"[Test/345/836] Permission denied for publishing (RPC error %d).",
		codes.PermissionDenied)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)

	membership := mock_au10.NewMockMembership(test.mock)
	membership.EXPECT().IsAllowed(test.rights).Return(false)

	publisher := mock_au10.NewMockPublisher(test.mock)
	publisher.EXPECT().GetMembership().Return(membership)

	test.service.EXPECT().GetPublisher().Return(publisher)

	request := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{Location: &proto.GeoPoint{Latitude: 9193.42}}}
	test.convert.EXPECT().
		VocalDeclarationFromProto(request, test.user, gomock.Any()).
		Return(&au10.VocalDeclaration{})

	response, err := test.client.AddVocal(request)
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = PermissionDenied desc = PERMISSION_DENIED")
}

func Test_Accesspoint_Client_AddVocal_ConvertError(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	test.log.EXPECT().Error(
		`[Test/345/836] Failed to convert vocal declaraton: "test error" (RPC error %d).`,
		codes.Internal)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)

	request := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{
			Location: &proto.GeoPoint{Latitude: 5935.63}}}
	test.convert.EXPECT().
		VocalDeclarationFromProto(request, test.user, gomock.Any()).
		Do(func(source *proto.VocalAddRequest,
			user au10.User,
			err *error) {
			test.assert.Nil(*err)
			*err = errors.New("test error")
		}).
		Return(nil)

	response, err := test.client.AddVocal(request)
	test.assert.Nil(response)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")
}

func Test_Accesspoint_Client_AddVocal_Execution(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	publisherMembership := mock_au10.NewMockMembership(test.mock)
	publisherMembership.EXPECT().IsAllowed(test.rights).Return(true)

	request := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{
			Location: &proto.GeoPoint{Latitude: 987.123}}}
	vocalDeclaration := &au10.VocalDeclaration{}
	test.convert.EXPECT().
		VocalDeclarationFromProto(request, test.user, gomock.Any()).
		Return(vocalDeclaration)

	vocal := mock_au10.NewMockVocal(test.mock)
	vocal.EXPECT().GetID().Return(au10.PostID(987))
	response := &proto.Vocal{}
	test.convert.EXPECT().
		VocalToProto(vocal, gomock.Any()).
		Return(response)

	publisher := mock_au10.NewMockPublisher(test.mock)
	publisher.EXPECT().GetMembership().Return(publisherMembership)
	publisher.EXPECT().AddVocal(vocalDeclaration).
		Return(vocal, nil)
	test.service.EXPECT().GetPublisher().Return(publisher)

	test.log.EXPECT().Debug(`[Test/345/836] Added vocal %d.`, au10.PostID(987))

	actualResponse, err := test.client.AddVocal(request)
	test.assert.NoError(err)
	test.assert.True(actualResponse == response)
}

func Test_Accesspoint_Client_AddVocal_ExecutionUnknownResult(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	request := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{
			Location: &proto.GeoPoint{Latitude: 987.123}}}
	vocalDeclaration := &au10.VocalDeclaration{}
	test.convert.EXPECT().
		VocalDeclarationFromProto(request, test.user, gomock.Any()).
		Return(vocalDeclaration)

	vocal := mock_au10.NewMockVocal(test.mock)
	vocal.EXPECT().GetID().Return(au10.PostID(987))
	test.convert.EXPECT().
		VocalToProto(vocal, gomock.Any()).
		Do(func(source au10.Vocal, err *error) {
			test.assert.Nil(*err)
			*err = errors.New("test error")
		}).
		Return(&proto.Vocal{})

	publisherMembership := mock_au10.NewMockMembership(test.mock)
	publisherMembership.EXPECT().IsAllowed(test.rights).Return(true)
	publisher := mock_au10.NewMockPublisher(test.mock)
	publisher.EXPECT().GetMembership().Return(publisherMembership)
	publisher.EXPECT().AddVocal(gomock.Any()).Return(vocal, nil)
	test.service.EXPECT().GetPublisher().Return(publisher)

	test.log.EXPECT().Error(
		`[Test/345/836] Failed to convert vocal: "test error" (RPC error %d).`,
		codes.Internal).
		After(test.log.EXPECT().Debug(
			`[Test/345/836] Added vocal %d.`, au10.PostID(987)))
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)

	actualResponse, err := test.client.AddVocal(request)
	test.assert.Nil(actualResponse)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")
}

func Test_Accesspoint_Client_AddVocal_ExecutionError(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	test.log.EXPECT().Error(
		`[Test/345/836] Failed to add: "test error" (RPC error %d).`,
		codes.Internal)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)

	membership := mock_au10.NewMockMembership(test.mock)
	membership.EXPECT().IsAllowed(test.rights).Return(true)

	publisher := mock_au10.NewMockPublisher(test.mock)
	publisher.EXPECT().GetMembership().Return(membership)
	publisher.EXPECT().AddVocal(gomock.Any()).
		Return(nil, errors.New("test error"))
	test.service.EXPECT().GetPublisher().Return(publisher)

	request := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{
			Location: &proto.GeoPoint{Latitude: 2987.123}}}
	vocalDeclaration := &au10.VocalDeclaration{}
	test.convert.EXPECT().
		VocalDeclarationFromProto(request, test.user, gomock.Any()).
		Return(vocalDeclaration)

	response, err := test.client.AddVocal(request)
	test.assert.Nil(response)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")
}

func Test_Accesspoint_Client_WriteMessageChunk_PermissionDenied(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	test.log.EXPECT().Error(
		`[Test/345/836] Permission denied for message "123"/"456" publishing (RPC error %d).`,
		codes.PermissionDenied)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)

	test.service.EXPECT().GetGlobalProps().MinTimes(1).Return(
		&proto.Props{MaxChunkSize: 9999})

	publisherMembership := mock_au10.NewMockMembership(test.mock)
	publisherMembership.EXPECT().IsAllowed(test.rights).Return(false)
	publisher := mock_au10.NewMockPublisher(test.mock)
	publisher.EXPECT().GetMembership().Return(publisherMembership)
	test.service.EXPECT().GetPublisher().Return(publisher)

	response, err := test.client.WriteMessageChunk(
		&proto.MessageChunkWriteRequest{
			PostID: "123", MessageID: "456", Chunk: []byte("test data")})

	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = PermissionDenied desc = PERMISSION_DENIED")
}

func Test_Accesspoint_Client_WriteMessageChunk_InvalidArgument(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	test.service.EXPECT().GetGlobalProps().MinTimes(1).
		Return(&proto.Props{MaxChunkSize: 1})
	test.logMembership.EXPECT().IsAllowed(test.rights).MinTimes(1).Return(false)
	publisherMembership := mock_au10.NewMockMembership(test.mock)
	publisherMembership.EXPECT().IsAllowed(test.rights).MinTimes(1).Return(true)
	publisher := mock_au10.NewMockPublisher(test.mock)
	publisher.EXPECT().GetMembership().MinTimes(1).Return(publisherMembership)
	test.service.EXPECT().GetPublisher().MinTimes(1).Return(publisher)

	test.user.EXPECT().BlockByProtocolMismatch()
	test.log.EXPECT().Error(
		`[Test/345/836] Failed to parse post ID "x"/"y": "test error 1" (RPC error %d).`,
		codes.InvalidArgument)
	request := &proto.MessageChunkWriteRequest{
		PostID: "x", MessageID: "y", Chunk: []byte("T")}
	test.convert.EXPECT().PostIDFromProto(request.PostID, gomock.Any()).Do(
		func(source string, err *error) {
			test.assert.Nil(*err)
			*err = errors.New("test error 1")
		}).
		Return(au10.PostID(1111))
	response, err := test.client.WriteMessageChunk(request)
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = INVALID_ARGUMENT")

	test.user.EXPECT().BlockByProtocolMismatch()
	test.log.EXPECT().Error(
		`[Test/345/836] Failed to parse message ID "x"/"y": "test error 2" (RPC error %d).`,
		codes.InvalidArgument)
	test.convert.EXPECT().PostIDFromProto(request.PostID, gomock.Any()).
		Return(au10.PostID(1111))
	test.convert.EXPECT().MessageIDFromProto(request.MessageID, gomock.Any()).Do(
		func(source string, err *error) {
			test.assert.Nil(*err)
			*err = errors.New("test error 2")
		}).
		Return(au10.MessageID(2222))
	response, err = test.client.WriteMessageChunk(request)
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = INVALID_ARGUMENT")

	request.Chunk = []byte("12")
	test.user.EXPECT().BlockByProtocolMismatch()
	test.log.EXPECT().Error(
		`[Test/345/836] Chunk size for message "x"/"y" is too big: 2 > 1 (RPC error %d).`,
		codes.InvalidArgument)
	response, err = test.client.WriteMessageChunk(request)
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = INVALID_ARGUMENT")

	request.Chunk = []byte{}
	test.user.EXPECT().BlockByProtocolMismatch()
	test.log.EXPECT().Error(
		`[Test/345/836] Chunk size is empty for message "x"/"y" (RPC error %d).`,
		codes.InvalidArgument)
	response, err = test.client.WriteMessageChunk(request)
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = INVALID_ARGUMENT")
}

func Test_Accesspoint_Client_WriteMessageChunk_Execution(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	test.service.EXPECT().GetGlobalProps().MinTimes(1).Return(
		&proto.Props{MaxChunkSize: 9999})

	publisherMembership := mock_au10.NewMockMembership(test.mock)
	publisherMembership.EXPECT().IsAllowed(test.rights).MinTimes(1).Return(true)
	publisher := mock_au10.NewMockPublisher(test.mock)
	publisher.EXPECT().GetMembership().MinTimes(1).Return(publisherMembership)
	test.service.EXPECT().GetPublisher().MinTimes(1).Return(publisher)

	request := &proto.MessageChunkWriteRequest{
		PostID: "x", MessageID: "y", Chunk: []byte("Test data")}
	test.convert.EXPECT().PostIDFromProto(request.PostID, gomock.Any()).
		MinTimes(1).
		Return(au10.PostID(1111))
	test.convert.EXPECT().MessageIDFromProto(request.MessageID, gomock.Any()).
		MinTimes(1).
		Return(au10.MessageID(2222))

	test.log.EXPECT().Debug(
		`[Test/345/836] Written %d bytes for "%s"/"%s".`,
		len(request.Chunk), "x", "y")
	publisher.EXPECT().AppendMessage(
		au10.MessageID(2222), au10.PostID(1111), test.user, request.Chunk).
		Return(nil)
	response, err := test.client.WriteMessageChunk(request)
	test.assert.NoError(err)
	test.assert.Equal(&proto.MessageChunkWriteResponse{}, response)

	test.log.EXPECT().Error(
		`[Test/345/836] Failed to write 9 bytes for "x"/"y": "test error" (RPC error %d).`,
		codes.Internal)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	publisher.EXPECT().AppendMessage(
		au10.MessageID(2222), au10.PostID(1111), test.user, request.Chunk).
		Return(errors.New("test error"))
	response, err = test.client.WriteMessageChunk(request)
	test.assert.Nil(response)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")
}

func Test_Accesspoint_Client_ReadMessage_InvalidArgument(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	test.logMembership.EXPECT().IsAllowed(test.rights).MinTimes(1).Return(false)

	test.user.EXPECT().BlockByProtocolMismatch()
	test.log.EXPECT().Error(
		`[Test/345/836] Failed to parse post ID "x"/"y": "test error 1" (RPC error %d).`,
		codes.InvalidArgument)
	request := &proto.MessageReadRequest{PostID: "x", MessageID: "y"}
	test.convert.EXPECT().PostIDFromProto(request.PostID, gomock.Any()).Do(
		func(source string, err *error) {
			test.assert.Nil(*err)
			*err = errors.New("test error 1")
		}).
		Return(au10.PostID(1111))
	err := test.client.ReadMessage(request, nil)
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = INVALID_ARGUMENT")

	test.user.EXPECT().BlockByProtocolMismatch()
	test.log.EXPECT().Error(
		`[Test/345/836] Failed to parse message ID "x"/"y": "test error 2" (RPC error %d).`,
		codes.InvalidArgument)
	test.convert.EXPECT().PostIDFromProto(request.PostID, gomock.Any()).
		Return(au10.PostID(1111))
	test.convert.EXPECT().MessageIDFromProto(request.MessageID, gomock.Any()).Do(
		func(source string, err *error) {
			test.assert.Nil(*err)
			*err = errors.New("test error 2")
		}).
		Return(au10.MessageID(2222))
	err = test.client.ReadMessage(request, nil)
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = INVALID_ARGUMENT")
}

func Test_Accesspoint_Client_GetAllowedMethods(t *testing.T) {
	test := newClientTest(t)
	defer test.close()

	postsMembership := mock_au10.NewMockMembership(test.mock)
	posts := mock_au10.NewMockPosts(test.mock)
	posts.EXPECT().GetMembership().MinTimes(1).Return(postsMembership)
	test.service.EXPECT().GetPosts().MinTimes(1).Return(posts)

	publisherMembership := mock_au10.NewMockMembership(test.mock)
	publisher := mock_au10.NewMockPublisher(test.mock)
	publisher.EXPECT().GetMembership().MinTimes(1).Return(publisherMembership)
	test.service.EXPECT().GetPublisher().MinTimes(1).Return(publisher)

	test.logMembership.EXPECT().IsAllowed(test.rights).Return(true)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(true)
	publisherMembership.EXPECT().IsAllowed(test.rights).Return(true)
	response := test.client.GetAllowedMethods()
	test.assert.True(response.ReadLog)
	test.assert.True(response.ReadPosts)
	test.assert.True(response.AddVocal)
	testNumberOfProtoStructFields(test.assert, response, 3)

	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(true)
	publisherMembership.EXPECT().IsAllowed(test.rights).Return(true)
	response = test.client.GetAllowedMethods()
	test.assert.False(response.ReadLog)
	test.assert.True(response.ReadPosts)
	test.assert.True(response.AddVocal)

	test.logMembership.EXPECT().IsAllowed(test.rights).Return(true)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(false)
	publisherMembership.EXPECT().IsAllowed(test.rights).Return(true)
	response = test.client.GetAllowedMethods()
	test.assert.True(response.ReadLog)
	test.assert.False(response.ReadPosts)
	test.assert.True(response.AddVocal)

	test.logMembership.EXPECT().IsAllowed(test.rights).Return(true)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(true)
	publisherMembership.EXPECT().IsAllowed(test.rights).Return(false)
	response = test.client.GetAllowedMethods()
	test.assert.True(response.ReadLog)
	test.assert.True(response.ReadPosts)
	test.assert.False(response.AddVocal)

}
