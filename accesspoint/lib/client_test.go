package accesspoint_test

import (
	"context"
	"errors"
	fmt "fmt"
	"reflect"
	"testing"
	"time"

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

	accesspoint := mock_ap.NewMockService(mock)
	user := mock_au10.NewMockUser(mock)

	client := ap.CreateClient(123, "Test", user, accesspoint)

	log := mock_au10.NewMockLog(mock)
	service := mock_au10.NewMockService(mock)
	service.EXPECT().Log().Times(4).Return(log)
	accesspoint.EXPECT().GetAu10().Times(4).Return(service)

	user.EXPECT().GetLogin().Return("error login")
	log.EXPECT().Error("[Test.123.error login] test error record %s", "argument")
	client.LogError("test error record %s", "argument")

	user.EXPECT().GetLogin().Return("warn login")
	log.EXPECT().Warn("[Test.123.warn login] test warn record %s", "argument")
	client.LogWarn("test warn record %s", "argument")

	user.EXPECT().GetLogin().Return("info login")
	log.EXPECT().Info("[Test.123.info login] test info record %s", "argument")
	client.LogInfo("test info record %s", "argument")

	user.EXPECT().GetLogin().Return("debug login")
	log.EXPECT().Debug("[Test.123.debug login] test debug record %s", "argument")
	client.LogDebug("test debug record %s", "argument")
}

type clientTest struct {
	mock          *gomock.Controller
	assert        *assert.Assertions
	service       *mock_au10.MockService
	accesspoint   *mock_ap.MockService
	log           *mock_au10.MockLog
	user          *mock_au10.MockUser
	logMembership *mock_au10.MockMembership
	rights        []au10.Rights
	client        ap.Client
}

func createClientTest(test *testing.T) *clientTest {
	result := &clientTest{
		mock:   gomock.NewController(test),
		assert: assert.New(test),
		rights: []au10.Rights{
			au10.CreateRights("x1", "z1"),
			au10.CreateRights("x2", "z2")}}
	result.service = mock_au10.NewMockService(result.mock)
	result.accesspoint = mock_ap.NewMockService(result.mock)
	result.log = mock_au10.NewMockLog(result.mock)
	result.user = mock_au10.NewMockUser(result.mock)
	result.logMembership = mock_au10.NewMockMembership(result.mock)
	result.client = ap.CreateClient(345, "Test", result.user, result.accesspoint)

	result.accesspoint.EXPECT().GetAu10().AnyTimes().Return(result.service)
	result.service.EXPECT().Log().AnyTimes().Return(result.log)
	result.log.EXPECT().GetMembership().AnyTimes().Return(result.logMembership)
	result.user.EXPECT().GetLogin().AnyTimes().Return("test login")
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
			"[Test.345.test login] Permission denied for %s (RPC error %%d).",
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
		`[Test.345.test login] Failed to subscribe: "test subscribe error" (RPC error %d).`,
		codes.Internal)
	response, err := run()
	test.assert.Nil(response)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")

}

func (test *clientTest) testNumberOfStructFields(
	s interface{}, numberOfFields int) {
	test.assert.Equal(numberOfFields,
		reflect.Indirect(reflect.ValueOf(s)).NumField())
}

func (test *clientTest) testNumberOfProtoStructFields(
	s interface{}, numberOfFields int) {
	test.testNumberOfStructFields(s, numberOfFields+3)
}

func (test *clientTest) expectPostConvertion() {}

func (test *clientTest) expectVocalConvertion(
	secondMessageKind au10.MessageKind) *mock_au10.MockVocal {
	result := mock_au10.NewMockVocal(test.mock)
	result.EXPECT().GetID().Return(au10.PostID(987))
	result.EXPECT().GetTime().Return(time.Unix(0, 567))
	result.EXPECT().GetLocation().Return(
		&au10.GeoPoint{Latitude: 123.456, Longitude: 456.789})
	message1 := mock_au10.NewMockMessage(test.mock)
	message1.EXPECT().GetID().Return(au10.MessageID(890))
	message1.EXPECT().GetKind().Return(au10.MessageKindText)
	message1.EXPECT().GetSize().Return(uint64(564))
	message2 := mock_au10.NewMockMessage(test.mock)
	message2.EXPECT().GetID().Return(au10.MessageID(892))
	message2.EXPECT().GetKind().Return(secondMessageKind)
	message2.EXPECT().GetSize().Return(uint64(565))
	result.EXPECT().GetMessages().Return([]au10.Message{message1, message2})
	return result
}

func (test *clientTest) validateExpectedVocalConvertion(vocal *proto.Vocal) {
	test.assert.NotNil(vocal)
	test.assert.Equal("987", vocal.Post.Id)
	test.assert.Equal(int64(567), vocal.Post.Time)
	test.assert.Equal(123.456, vocal.Post.Location.Latitude)
	test.assert.Equal(456.789, vocal.Post.Location.Longitude)
	test.testNumberOfProtoStructFields(vocal.Post.Location, 2)
	test.assert.Equal([]*proto.Message{
		&proto.Message{
			Id: "890", Kind: proto.Message_TEXT, Size: uint64(564)},
		&proto.Message{
			Id: "892", Kind: proto.Message_TEXT, Size: uint64(565)}},
		vocal.Post.Messages)
	test.assert.Equal(1, len(proto.Message_Kind_value))
	test.testNumberOfProtoStructFields(vocal.Post.Messages[0], 3)
	test.testNumberOfProtoStructFields(vocal.Post, 4)
	test.testNumberOfProtoStructFields(vocal, 1)
}

func (test *clientTest) testSubscription(
	prepare func(*ap.SubscriptionInfo),
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
	test.testNumberOfStructFields(subscriptionInfo, 1)

	test.log.EXPECT().Debug("[Test.345.test login] Canceled (%d/%d).",
		uint32(101), uint32(909)).
		After(test.log.EXPECT().Debug("[Test.345.test login] Subscribed (%d/%d).",
			uint32(102), uint32(808)))

	prepare(subscriptionInfo)

	test.accesspoint.EXPECT().UnregisterSubscriber().Return(uint32(909)).After(
		test.accesspoint.EXPECT().RegisterSubscriber().Return(uint32(808)))

	go func() {
		createData()
		ctxDoneChan <- struct{}{}
	}()
	test.assert.NoError(run(ctx, false))

	// Failed execution

	subscriptionInfo.NumberOfSubscribers = 201

	test.log.EXPECT().Error(
		`[Test.345.test login] Failed: "test subscription error" (201/2909) (RPC error %d).`,
		codes.Internal).
		After(test.log.EXPECT().Debug("[Test.345.test login] Subscribed (%d/%d).",
			uint32(202), uint32(2808)))

	prepare(subscriptionInfo)

	test.accesspoint.EXPECT().UnregisterSubscriber().Return(uint32(2909)).After(
		test.accesspoint.EXPECT().RegisterSubscriber().Return(uint32(2808)))

	go func() {
		createData()
		test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
		createErr(errors.New("test subscription error"))
	}()
	test.assert.EqualError(run(ctx, false),
		"rpc error: code = Internal desc = INTERNAL")

	// Closed

	subscriptionInfo.NumberOfSubscribers = 301

	test.log.EXPECT().Debug("[Test.345.test login] Canceled (%d/%d).",
		uint32(301), uint32(3909)).
		After(test.log.EXPECT().Debug("[Test.345.test login] Subscribed (%d/%d).",
			uint32(302), uint32(3808)))

	prepare(subscriptionInfo)

	test.accesspoint.EXPECT().UnregisterSubscriber().Return(uint32(3909)).After(
		test.accesspoint.EXPECT().RegisterSubscriber().Return(uint32(3808)))

	go func() {
		createData()
		closeSubscription()
	}()
	test.assert.NoError(run(ctx, false))

	// Data with error

	subscriptionInfo.NumberOfSubscribers = 401

	for _, dataMock := range createErrorData {

		test.log.EXPECT().Error(
			`[Test.345.test login] Failed: "`+dataMock.errText+`" (401/4909) (RPC error %d).`,
			codes.Internal).
			After(test.log.EXPECT().Debug("[Test.345.test login] Subscribed (%d/%d).",
				uint32(402), uint32(4808)))

		prepare(subscriptionInfo)

		test.accesspoint.EXPECT().UnregisterSubscriber().Return(uint32(4909)).After(
			test.accesspoint.EXPECT().RegisterSubscriber().Return(uint32(4808)))

		go func() {
			test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
			dataMock.run()
		}()
		test.assert.EqualError(run(ctx, true),
			"rpc error: code = Internal desc = INTERNAL")

	}
}

func (test *clientTest) prepareMessageExecution() *mock_au10.MockMessage {
	messageMembership := mock_au10.NewMockMembership(test.mock)
	messageMembership.EXPECT().IsAllowed(test.rights).MinTimes(1).Return(true)

	message := mock_au10.NewMockMessage(test.mock)
	message.EXPECT().GetID().MinTimes(1).Return(au10.MessageID(456))
	message.EXPECT().GetMembership().MinTimes(1).Return(messageMembership)

	postMembership := mock_au10.NewMockMembership(test.mock)
	postMembership.EXPECT().IsAllowed(test.rights).MinTimes(1).Return(true)

	post := mock_au10.NewMockPost(test.mock)
	post.EXPECT().GetMembership().MinTimes(1).Return(postMembership)
	post.EXPECT().GetMessages().MinTimes(1).Return([]au10.Message{message})

	postsMembership := mock_au10.NewMockMembership(test.mock)
	postsMembership.EXPECT().IsAllowed(test.rights).MinTimes(1).Return(true)

	posts := mock_au10.NewMockPosts(test.mock)
	posts.EXPECT().GetMembership().MinTimes(1).Return(postsMembership)
	posts.EXPECT().GetPost(au10.PostID(123)).MinTimes(1).Return(post, nil)

	test.service.EXPECT().GetPosts().MinTimes(1).Return(posts)

	return message
}

func testClientMessagePermissionDenied(
	t *testing.T,
	action string,
	run func(client ap.Client, postID, messageID string) (interface{}, error)) {

	test := createClientTest(t)
	defer test.close()

	test.log.EXPECT().Error(
		`[Test.345.test login] Permission denied for message "123"/"456" (RPC error %d).`,
		codes.PermissionDenied)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	messageMembership := mock_au10.NewMockMembership(test.mock)
	messageMembership.EXPECT().IsAllowed(test.rights).Return(false)
	message := mock_au10.NewMockMessage(test.mock)
	message.EXPECT().GetID().Return(au10.MessageID(456))
	message.EXPECT().GetMembership().Return(messageMembership)
	postMembership := mock_au10.NewMockMembership(test.mock)
	postMembership.EXPECT().IsAllowed(test.rights).Return(true)
	post := mock_au10.NewMockPost(test.mock)
	post.EXPECT().GetMembership().Return(postMembership)
	post.EXPECT().GetMessages().Return([]au10.Message{message})
	postsMembership := mock_au10.NewMockMembership(test.mock)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(true)
	posts := mock_au10.NewMockPosts(test.mock)
	posts.EXPECT().GetMembership().Return(postsMembership)
	posts.EXPECT().GetPost(au10.PostID(123)).Return(post, nil)
	test.service.EXPECT().GetPosts().Return(posts)
	response, err := run(test.client, "123", "456")
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = PermissionDenied desc = PERMISSION_DENIED")

	test.log.EXPECT().Error(
		`[Test.345.test login] Permission denied for post "123"/"456" (RPC error %d).`,
		codes.PermissionDenied)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	postMembership.EXPECT().IsAllowed(test.rights).Return(false)
	post.EXPECT().GetMembership().Return(postMembership)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(true)
	posts.EXPECT().GetMembership().Return(postsMembership)
	posts.EXPECT().GetPost(au10.PostID(123)).Return(post, nil)
	test.service.EXPECT().GetPosts().Return(posts)
	response, err = run(test.client, "123", "456")
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = PermissionDenied desc = PERMISSION_DENIED")

	test.log.EXPECT().Error(
		"[Test.345.test login] Permission denied for posts (RPC error %d).",
		codes.PermissionDenied)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(false)
	posts.EXPECT().GetMembership().Return(postsMembership)
	test.service.EXPECT().GetPosts().Return(posts)
	response, err = run(test.client, "123", "456")
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = PermissionDenied desc = PERMISSION_DENIED")
}

func testClientMessageInvalidArgument(
	t *testing.T,
	action string,
	run func(client ap.Client, postID, messageID string) (interface{}, error)) {

	test := createClientTest(t)
	defer test.close()

	postsMembership := mock_au10.NewMockMembership(test.mock)
	postsMembership.EXPECT().IsAllowed(test.rights).MinTimes(1).Return(true)
	posts := mock_au10.NewMockPosts(test.mock)
	posts.EXPECT().GetMembership().MinTimes(1).Return(postsMembership)
	test.service.EXPECT().GetPosts().MinTimes(1).Return(posts)
	test.logMembership.EXPECT().IsAllowed(test.rights).MinTimes(1).Return(false)

	test.log.EXPECT().Error(
		`[Test.345.test login] Failed to parse post ID "---"/"456": "strconv.ParseUint: parsing "---": invalid syntax" (RPC error %d).`,
		codes.InvalidArgument)
	response, err := run(test.client, "---", "456")
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = INVALID_ARGUMENT")

	test.log.EXPECT().Error(
		`[Test.345.test login] Failed to get post "123"/"456": "test error" (RPC error %d).`,
		codes.InvalidArgument)
	posts.EXPECT().GetPost(au10.PostID(123)).Return(nil, errors.New("test error"))
	response, err = run(test.client, "123", "456")
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = INVALID_ARGUMENT")

	test.log.EXPECT().Error(
		`[Test.345.test login] Failed to parse message ID "123"/"abc": "strconv.ParseUint: parsing "abc": invalid syntax" (RPC error %d).`,
		codes.InvalidArgument)
	response, err = run(test.client, "123", "abc")
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = INVALID_ARGUMENT")

	test.log.EXPECT().Error(
		`[Test.345.test login] Failed to find message "123"/"456" (RPC error %d).`,
		codes.InvalidArgument)
	message := mock_au10.NewMockMessage(test.mock)
	message.EXPECT().GetID().Return(au10.MessageID(987))
	postMembership := mock_au10.NewMockMembership(test.mock)
	postMembership.EXPECT().IsAllowed(test.rights).Return(true)
	post := mock_au10.NewMockPost(test.mock)
	post.EXPECT().GetMembership().Return(postMembership)
	post.EXPECT().GetMessages().Return([]au10.Message{message})
	posts.EXPECT().GetPost(au10.PostID(123)).Return(post, nil)
	response, err = run(test.client, "123", "456")
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = INVALID_ARGUMENT")
}

func Test_Accesspoint_Client_CreateError(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	test.log.EXPECT().Error("[Test.345.test login] Test error full text (RPC error %d).",
		codes.Internal)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	err := test.client.CreateError(codes.Internal, "test error full text")
	test.assert.EqualError(err,
		"rpc error: code = Internal desc = INTERNAL")

	test.log.EXPECT().Error("[Test.345.test login] Test error full text 2 (RPC error %d).",
		codes.InvalidArgument)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(true)
	err = test.client.CreateError(codes.InvalidArgument, "test error full text 2")
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = test error full text 2")

	test.log.EXPECT().Error("[Test.345.test login] RPC error %d.", codes.Code(18))
	test.log.EXPECT().Error(
		"[Test.345.test login] Failed to find external description for status code %d.",
		codes.Code(18))
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	err = test.client.CreateError(codes.Code(18), "")
	test.assert.EqualError(err, "rpc error: code = Code(18) desc = unknown error")
}

func Test_Accesspoint_Client_Auth(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	test.testNumberOfProtoStructFields(&proto.AuthRequest{}, 1)

	users := mock_au10.NewMockUsers(test.mock)
	test.service.EXPECT().GetUsers().Times(3).Return(users)

	user := mock_au10.NewMockUser(test.mock)

	testToken := "test token"
	users.EXPECT().Auth("test login x").
		Return(user, &testToken, errors.New("test error"))
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	test.log.EXPECT().Error(
		`[Test.345.test login] Failed to auth "test login x": "test error" (RPC error %d).`,
		codes.Internal)
	token, err := test.client.Auth(&proto.AuthRequest{Login: "test login x"})
	test.assert.Nil(token)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")

	users.EXPECT().Auth("test login x2").Return(user, nil, nil)
	test.log.EXPECT().Info(
		`[Test.345.test login] Wrong creds for "%s"`, "test login x2")
	token, err = test.client.Auth(&proto.AuthRequest{Login: "test login x2"})
	test.assert.Nil(token)
	test.assert.NoError(err)

	users.EXPECT().Auth("test login x3").Return(user, &testToken, nil)
	user.EXPECT().GetLogin().Return("test login x4")
	test.log.EXPECT().Info(`[Test.345.test login x4] Authed "%s".`, "test login x3")
	token, err = test.client.Auth(&proto.AuthRequest{Login: "test login x3"})
	test.assert.NotNil(token)
	test.assert.Equal("test token", *token)
	test.assert.NoError(err)
}

func Test_Accesspoint_Client_ReadLog(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	response := mock_proto.NewMockAu10_ReadLogServer(test.mock)

	test.testSubscribeError(
		test.logMembership,
		func(err error) { test.log.EXPECT().Subscribe().Return(nil, err) },
		func() (interface{}, error) {
			return nil, test.client.ReadLog(nil, response)
		},
		"log")

	var recordsChan chan au10.LogRecord
	var errChan chan error
	test.testSubscription(
		func(subscriptionInfo *ap.SubscriptionInfo) {
			test.accesspoint.EXPECT().GetLogSubscriptionInfo().
				Return(subscriptionInfo)
		},
		func(ctx context.Context, isErrData bool) error {
			test.assert.False(isErrData)
			subscription := mock_au10.NewMockLogSubscription(test.mock)
			errChan = make(chan error)
			subscription.EXPECT().GetErrChan().MinTimes(1).Return(errChan)
			recordsChan = make(chan au10.LogRecord)
			subscription.EXPECT().GetRecordsChan().MinTimes(1).Return(recordsChan)
			subscription.EXPECT().Close().Do(func() {
				if recordsChan != nil {
					close(recordsChan)
				}
				close(errChan)
			})
			test.log.EXPECT().Subscribe().Return(subscription, nil)
			test.logMembership.EXPECT().IsAllowed(test.rights).Return(true)
			response.EXPECT().Send(gomock.Any()).Do(func(record *proto.LogRecord) {
				test.assert.Equal(int64(987), record.SeqNum)
				test.assert.Equal(int64(567), record.Time)
				test.assert.Equal("test log record", record.Text)
				test.assert.Equal("debug", record.Severity)
				test.assert.Equal("test node type", record.NodeType)
				test.assert.Equal("test node name", record.NodeName)
				test.testNumberOfProtoStructFields(record, 6)
			}).Return(nil)
			response.EXPECT().Context().Return(ctx)
			return test.client.ReadLog(nil, response)
		},
		func() {
			record := mock_au10.NewMockLogRecord(test.mock)
			record.EXPECT().GetSequenceNumber().Return(int64(987))
			record.EXPECT().GetTime().Return(time.Unix(0, 567))
			record.EXPECT().GetText().Return("test log record")
			record.EXPECT().GetSeverity().Return("debug")
			record.EXPECT().GetNodeType().Return("test node type")
			record.EXPECT().GetNodeName().Return("test node name")
			recordsChan <- record
		},
		nil,
		func(err error) { errChan <- err },
		func() {
			close(recordsChan)
			recordsChan = nil
		})
}

func Test_Accesspoint_Client_ReadPosts(t *testing.T) {
	test := createClientTest(t)
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

	var postsChan chan au10.Post
	var errChan chan error
	test.testSubscription(
		func(subscriptionInfo *ap.SubscriptionInfo) {
			test.accesspoint.EXPECT().GetPostsSubscriptionInfo().
				Return(subscriptionInfo)
		},
		func(ctx context.Context, isErrData bool) error {
			subscription := mock_au10.NewMockPostsSubscription(test.mock)
			errChan = make(chan error)
			subscription.EXPECT().GetErrChan().MinTimes(1).Return(errChan)
			postsChan = make(chan au10.Post)
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
				response.EXPECT().Send(gomock.Any()).Do(func(post *proto.PostUpdate) {
					test.assert.IsType((*proto.PostUpdate_Vocal)(nil), post.Post)
					test.validateExpectedVocalConvertion(
						post.Post.(*proto.PostUpdate_Vocal).Vocal)
					test.testNumberOfProtoStructFields(post, 1)
				}).Return(nil)
			}
			response.EXPECT().Context().Return(ctx)
			return test.client.ReadPosts(nil, response)
		},
		func() { postsChan <- test.expectVocalConvertion(au10.MessageKindText) },
		[]struct {
			errText string
			run     func()
		}{
			struct {
				errText string
				run     func()
			}{errText: `unknown post type "<nil>"`, run: func() { postsChan <- nil }},
			struct {
				errText string
				run     func()
			}{
				errText: "unknown au10 message kind 1",
				run:     func() { postsChan <- test.expectVocalConvertion(1) }}},
		func(err error) { errChan <- err },
		func() {
			close(postsChan)
			postsChan = nil
		})
}

func Test_Accesspoint_Client_AddVocal_PermissionDenied(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	test.log.EXPECT().Error(
		"[Test.345.test login] Permission denied for posts (RPC error %d).",
		codes.PermissionDenied)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)

	postsMembership := mock_au10.NewMockMembership(test.mock)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(false)

	posts := mock_au10.NewMockPosts(test.mock)
	posts.EXPECT().GetMembership().Return(postsMembership)

	test.service.EXPECT().GetPosts().Return(posts)

	emptyRequest := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{
			Messages: []*proto.PostAddRequest_MessageDeclaration{}}}

	response, err := test.client.AddVocal(emptyRequest)
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = PermissionDenied desc = PERMISSION_DENIED")
}

func Test_Accesspoint_Client_AddVocal_InvalidArgument(t *testing.T) {
	test := createClientTest(t)
	defer test.close()
	test.log.EXPECT().Error(
		`[Test.345.test login] Failed to convert message kind: "unknown proto message kind 1" (RPC error %d).`,
		codes.Internal)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	request := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{
			Messages: []*proto.PostAddRequest_MessageDeclaration{
				&proto.PostAddRequest_MessageDeclaration{Kind: proto.Message_TEXT},
				&proto.PostAddRequest_MessageDeclaration{Kind: 1}}}}
	response, err := test.client.AddVocal(request)
	test.assert.Nil(response)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")
}

func (test *clientTest) testAddPostExecution(
	expectAdd func(
		*mock_au10.MockPostsMockRecorder,
		[]au10.MessageDeclaration,
		au10.User) *gomock.Call,
	run func() (*proto.Post, error)) {

}

func Test_Accesspoint_Client_AddVocal_Execution(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	postsMembership := mock_au10.NewMockMembership(test.mock)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(true)

	posts := mock_au10.NewMockPosts(test.mock)
	posts.EXPECT().GetMembership().Return(postsMembership)
	posts.EXPECT().AddVocal(
		[]au10.MessageDeclaration{
			au10.MessageDeclaration{Kind: au10.MessageKindText, Size: 123},
			au10.MessageDeclaration{Kind: au10.MessageKindText, Size: 456}},
		test.user).Return(test.expectVocalConvertion(au10.MessageKindText), nil)
	test.service.EXPECT().GetPosts().Return(posts)

	request := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{
			Location: &proto.GeoPoint{Latitude: 123.456, Longitude: 456.789},
			Messages: []*proto.PostAddRequest_MessageDeclaration{
				&proto.PostAddRequest_MessageDeclaration{
					Kind: proto.Message_TEXT,
					Size: 123},
				&proto.PostAddRequest_MessageDeclaration{
					Kind: proto.Message_TEXT,
					Size: 456}}}}
	test.assert.Equal(1, len(proto.Message_Kind_name))
	test.testNumberOfProtoStructFields(request, 1)
	test.testNumberOfProtoStructFields(request.Post, 2)
	test.testNumberOfProtoStructFields(request.Post.Messages[0], 2)

	response, err := test.client.AddVocal(request)
	test.validateExpectedVocalConvertion(response)
	test.testNumberOfProtoStructFields(response, 1)
	test.assert.NoError(err)
}

func Test_Accesspoint_Client_AddVocal_ExecutionUnknownResult(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	postsMembership := mock_au10.NewMockMembership(test.mock)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(true)

	posts := mock_au10.NewMockPosts(test.mock)

	request := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{
			Location: &proto.GeoPoint{Latitude: 123.456, Longitude: 456.789},
			Messages: []*proto.PostAddRequest_MessageDeclaration{
				&proto.PostAddRequest_MessageDeclaration{
					Kind: proto.Message_TEXT,
					Size: 321},
				&proto.PostAddRequest_MessageDeclaration{
					Kind: proto.Message_TEXT,
					Size: 654}}}}

	posts.EXPECT().GetMembership().Return(postsMembership)
	posts.EXPECT().AddVocal(
		[]au10.MessageDeclaration{
			au10.MessageDeclaration{Kind: au10.MessageKindText, Size: 321},
			au10.MessageDeclaration{Kind: au10.MessageKindText, Size: 654}},
		test.user).Return(test.expectVocalConvertion(1), nil)
	test.service.EXPECT().GetPosts().Return(posts)

	test.log.EXPECT().Error(
		`[Test.345.test login] Failed to convert vocal: "unknown au10 message kind 1" (RPC error %d).`,
		codes.Internal)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)

	response, err := test.client.AddVocal(request)
	test.assert.Nil(response)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")
}

func Test_Accesspoint_Client_AddVocal_ExecutionError(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	test.log.EXPECT().Error(
		`[Test.345.test login] Failed to add: "test error" (RPC error %d).`,
		codes.Internal)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)

	postsMembership := mock_au10.NewMockMembership(test.mock)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(true)

	posts := mock_au10.NewMockPosts(test.mock)
	posts.EXPECT().GetMembership().Return(postsMembership)
	posts.EXPECT().AddVocal([]au10.MessageDeclaration{}, test.user).
		Return(nil, errors.New("test error"))
	test.service.EXPECT().GetPosts().Return(posts)

	request := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{
			Messages: []*proto.PostAddRequest_MessageDeclaration{}}}

	response, err := test.client.AddVocal(request)
	test.assert.Nil(response)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")
}

func Test_Accesspoint_Client_WriteMessageChunk_PermissionDenied(t *testing.T) {
	testClientMessagePermissionDenied(
		t,
		"write",
		func(client ap.Client, postID, messageID string) (interface{}, error) {
			return client.WriteMessageChunk(
				&proto.MessageChunkWriteRequest{PostID: postID, MessageID: messageID})
		})
}

func Test_Accesspoint_Client_WriteMessageChunk_InvalidArgument(t *testing.T) {
	testClientMessageInvalidArgument(
		t,
		"write",
		func(client ap.Client, postID, messageID string) (interface{}, error) {
			return client.WriteMessageChunk(
				&proto.MessageChunkWriteRequest{PostID: postID, MessageID: messageID})
		})
}

func Test_Accesspoint_Client_WriteMessageChunk_Execution(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	message := test.prepareMessageExecution()

	request := &proto.MessageChunkWriteRequest{
		PostID:    "123",
		MessageID: "456",
		Chunk:     []byte("some bytes")}
	test.testNumberOfProtoStructFields(request, 3)
	message.EXPECT().Append(request.Chunk).Return(nil)
	response, err := test.client.WriteMessageChunk(request)
	test.assert.NotNil(response)
	test.testNumberOfProtoStructFields(response, 0)
	test.assert.NoError(err)

	test.log.EXPECT().Error(
		`[Test.345.test login] Failed to write chunk "123"/"456": "test error" (RPC error %d).`,
		codes.Internal)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	message.EXPECT().Append(request.Chunk).Return(errors.New("test error"))
	response, err = test.client.WriteMessageChunk(request)
	test.assert.Nil(response)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")
}

func Test_Accesspoint_Client_ReadMessage_PermissionDenied(t *testing.T) {
	testClientMessagePermissionDenied(
		t,
		"read",
		func(client ap.Client, postID, messageID string) (interface{}, error) {
			return nil, client.ReadMessage(
				&proto.MessageReadRequest{PostID: postID, MessageID: messageID}, nil)
		})
}

func Test_Accesspoint_Client_ReadMessage_InvalidArgument(t *testing.T) {
	testClientMessageInvalidArgument(
		t,
		"read",
		func(client ap.Client, postID, messageID string) (interface{}, error) {
			return nil, client.ReadMessage(
				&proto.MessageReadRequest{PostID: postID, MessageID: messageID}, nil)
		})
}

func Test_Accesspoint_Client_ReadMessage_Execution(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	message := test.prepareMessageExecution()
	props := &proto.Props{MaxChunkSize: 3}
	test.accesspoint.EXPECT().GetGlobalProps().MinTimes(1).Return(props)

	request := &proto.MessageReadRequest{PostID: "123", MessageID: "456"}
	test.testNumberOfProtoStructFields(request, 2)

	message.EXPECT().GetSize().Return(uint64(7))
	message.EXPECT().Load(gomock.Any(), uint64(6)).Do(
		func(buffer *[]byte, offset uint64) {
			test.assert.Equal(props.MaxChunkSize, uint64(len(*buffer)))
			*buffer = []byte("6")
		}).
		Return(nil).
		After(message.EXPECT().Load(gomock.Any(), uint64(3)).Do(
			func(buffer *[]byte, offset uint64) {
				test.assert.Equal(props.MaxChunkSize, uint64(len(*buffer)))
				*buffer = []byte("345")
			}).
			Return(nil).After(
			message.EXPECT().Load(gomock.Any(), uint64(0)).Do(
				func(buffer *[]byte, offset uint64) {
					test.assert.Equal(props.MaxChunkSize, uint64(len(*buffer)))
					*buffer = []byte("012")
				}).
				Return(nil)))

	chunkIndexVar := 0
	chunkIndex := &chunkIndexVar
	stream := mock_proto.NewMockAu10_ReadMessageServer(test.mock)
	stream.EXPECT().Send(gomock.Any()).Times(3).
		Do(func(chunk *proto.MessageChunk) {
			switch *chunkIndex {
			case 0:
				test.assert.Equal([]byte("012"), chunk.Chunk)
			case 1:
				test.assert.Equal([]byte("345"), chunk.Chunk)
			case 2:
				test.assert.Equal([]byte("6"), chunk.Chunk)
			default:
				test.assert.Equal(0, *chunkIndex)
			}
			*chunkIndex = *chunkIndex + 1
		})

	err := test.client.ReadMessage(request, stream)
	test.assert.NoError(err)
}

func Test_Accesspoint_Client_ReadMessage_ExecutionErrors(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	message := test.prepareMessageExecution()
	props := &proto.Props{MaxChunkSize: 3}
	test.accesspoint.EXPECT().GetGlobalProps().MinTimes(1).Return(props)

	request := &proto.MessageReadRequest{PostID: "123", MessageID: "456"}
	test.testNumberOfProtoStructFields(request, 2)
	message.EXPECT().GetSize().MinTimes(1).Return(uint64(7))

	test.log.EXPECT().Error(
		`[Test.345.test login] Failed to load chunk "123"/"456"/3: "test load error" (RPC error %d).`,
		codes.Internal)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	message.EXPECT().Load(gomock.Any(), uint64(3)).
		Return(errors.New("test load error")).
		After(message.EXPECT().Load(gomock.Any(), uint64(0)).Return(nil))
	stream := mock_proto.NewMockAu10_ReadMessageServer(test.mock)
	stream.EXPECT().Send(gomock.Any()).Return(nil)
	err := test.client.ReadMessage(request, stream)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")

	test.log.EXPECT().Error(
		`[Test.345.test login] Failed to send chunk "123"/"456"/3: "test send error" (RPC error %d).`,
		codes.Internal)
	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	message.EXPECT().Load(gomock.Any(), uint64(3)).
		Return(nil).
		After(message.EXPECT().Load(gomock.Any(), uint64(0)).Return(nil))
	stream.EXPECT().Send(gomock.Any()).
		Return(errors.New("test send error")).
		After(stream.EXPECT().Send(gomock.Any()).Return(nil))
	err = test.client.ReadMessage(request, stream)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")
}

func Test_Accesspoint_Client_GetAllowedMethods(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	postsMembership := mock_au10.NewMockMembership(test.mock)
	posts := mock_au10.NewMockPosts(test.mock)
	posts.EXPECT().GetMembership().MinTimes(1).Return(postsMembership)
	test.service.EXPECT().GetPosts().MinTimes(1).Return(posts)

	test.logMembership.EXPECT().IsAllowed(test.rights).Return(true)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(true)
	response := test.client.GetAllowedMethods()
	test.assert.True(response.ReadLog)
	test.assert.True(response.ReadPosts)
	test.assert.True(response.AddVocal)
	test.testNumberOfProtoStructFields(response, 3)

	test.logMembership.EXPECT().IsAllowed(test.rights).Return(false)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(true)
	response = test.client.GetAllowedMethods()
	test.assert.False(response.ReadLog)
	test.assert.True(response.ReadPosts)
	test.assert.True(response.AddVocal)
	test.testNumberOfProtoStructFields(response, 3)

	test.logMembership.EXPECT().IsAllowed(test.rights).Return(true)
	postsMembership.EXPECT().IsAllowed(test.rights).Return(false)
	response = test.client.GetAllowedMethods()
	test.assert.True(response.ReadLog)
	test.assert.False(response.ReadPosts)
	test.assert.False(response.AddVocal)
	test.testNumberOfProtoStructFields(response, 3)
}
