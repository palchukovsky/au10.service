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

	client := ap.CreateClient(123, user, accesspoint)

	log := mock_au10.NewMockLog(mock)
	service := mock_au10.NewMockService(mock)
	service.EXPECT().Log().Times(4).Return(log)
	accesspoint.EXPECT().GetAu10().Times(4).Return(service)

	user.EXPECT().GetLogin().Return("error login")
	log.EXPECT().Error("123.error login: test error record %s", "argument")
	client.LogError("test error record %s", "argument")

	user.EXPECT().GetLogin().Return("warn login")
	log.EXPECT().Warn("123.warn login: test warn record %s", "argument")
	client.LogWarn("test warn record %s", "argument")

	user.EXPECT().GetLogin().Return("info login")
	log.EXPECT().Info("123.info login: test info record %s", "argument")
	client.LogInfo("test info record %s", "argument")

	user.EXPECT().GetLogin().Return("debug login")
	log.EXPECT().Debug("123.debug login: test debug record %s", "argument")
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
	result.client = ap.CreateClient(345, result.user, result.accesspoint)

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

	membership.EXPECT().IsAvailable(test.rights).Return(false)
	test.logMembership.EXPECT().IsAvailable(test.rights).Return(false)
	test.log.EXPECT().Error(
		fmt.Sprintf("345.test login: Permission denied to %s (RPC error %%d).",
			action),
		codes.PermissionDenied)
	response, err := run()
	test.assert.Nil(response)
	test.assert.EqualError(err,
		"rpc error: code = PermissionDenied desc = PERMISSION_DENIED")
}

func (test *clientTest) testSubscription(
	run func(ctx context.Context) error,
	createData func(),
	createErr func(error),
	closeSubscription func()) {

	// Successfull execution

	ctx := mock_context.NewMockContext(test.mock)
	ctxDoneChan := make(chan struct{})
	defer close(ctxDoneChan)
	ctx.EXPECT().Done().MinTimes(1).Return(ctxDoneChan)

	subscriptionInfo := &ap.SubscriptionInfo{
		Name:                "test subscription",
		NumberOfSubscribers: 101}
	test.assert.Equal(2,
		reflect.Indirect(reflect.ValueOf(subscriptionInfo)).NumField())

	test.log.EXPECT().Debug(
		"345.test login: Subscription to %s canceled (%d/%d).",
		subscriptionInfo.Name, uint32(101), uint32(909)).
		After(test.log.EXPECT().Debug("345.test login: Subscribed to %s (%d/%d).",
			subscriptionInfo.Name, uint32(102), uint32(808)))

	test.accesspoint.EXPECT().GetLogSubscriptionInfo().Return(subscriptionInfo)
	test.accesspoint.EXPECT().UnregisterSubscriber().Return(uint32(909)).After(
		test.accesspoint.EXPECT().RegisterSubscriber().Return(uint32(808)))

	go func() {
		createData()
		ctxDoneChan <- struct{}{}
	}()
	test.assert.NoError(run(ctx))

	// Failed execution

	subscriptionInfo.NumberOfSubscribers = 201

	test.log.EXPECT().Error(fmt.Sprintf(
		`345.test login: Subscription to %s error: "test subscription error" (%d/%d) (RPC error %%d).`,
		subscriptionInfo.Name, 201, 2909),
		codes.Internal).
		After(test.log.EXPECT().Debug("345.test login: Subscribed to %s (%d/%d).",
			subscriptionInfo.Name, uint32(202), uint32(2808)))

	test.accesspoint.EXPECT().GetLogSubscriptionInfo().Return(subscriptionInfo)
	test.accesspoint.EXPECT().UnregisterSubscriber().Return(uint32(2909)).After(
		test.accesspoint.EXPECT().RegisterSubscriber().Return(uint32(2808)))

	go func() {
		createData()
		test.logMembership.EXPECT().IsAvailable(test.rights).Return(false)
		createErr(errors.New("test subscription error"))
	}()
	test.assert.EqualError(run(ctx), "rpc error: code = Internal desc = INTERNAL")

	// Closed

	subscriptionInfo.NumberOfSubscribers = 301

	test.log.EXPECT().Debug(
		"345.test login: Subscription to %s canceled (%d/%d).",
		subscriptionInfo.Name, uint32(301), uint32(3909)).
		After(test.log.EXPECT().Debug("345.test login: Subscribed to %s (%d/%d).",
			subscriptionInfo.Name, uint32(302), uint32(3808)))

	test.accesspoint.EXPECT().GetLogSubscriptionInfo().Return(subscriptionInfo)
	test.accesspoint.EXPECT().UnregisterSubscriber().Return(uint32(3909)).After(
		test.accesspoint.EXPECT().RegisterSubscriber().Return(uint32(3808)))

	go func() {
		createData()
		closeSubscription()
	}()
	test.assert.NoError(run(ctx))
}

func Test_Accesspoint_Client_CreateError(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	test.log.EXPECT().Error("345.test login: Test error full text (RPC error %d).",
		codes.Internal)
	test.logMembership.EXPECT().IsAvailable(test.rights).Return(false)
	err := test.client.CreateError(codes.Internal, "test error full text")
	test.assert.EqualError(err,
		"rpc error: code = Internal desc = INTERNAL")

	test.log.EXPECT().Error("345.test login: Test error full text 2 (RPC error %d).",
		codes.InvalidArgument)
	test.logMembership.EXPECT().IsAvailable(test.rights).Return(true)
	err = test.client.CreateError(codes.InvalidArgument, "test error full text 2")
	test.assert.EqualError(err,
		"rpc error: code = InvalidArgument desc = test error full text 2")

	test.log.EXPECT().Error("345.test login: RPC error %d.", codes.Code(18))
	test.log.EXPECT().Error(
		"345.test login: Failed to find external description for status code %d.",
		codes.Code(18))
	test.logMembership.EXPECT().IsAvailable(test.rights).Return(false)
	err = test.client.CreateError(codes.Code(18), "")
	test.assert.EqualError(err, "rpc error: code = Code(18) desc = unknown error")
}

func Test_Accesspoint_Client_Auth(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	test.assert.Equal(1+3,
		reflect.Indirect(reflect.ValueOf(&proto.AuthRequest{})).NumField())

	users := mock_au10.NewMockUsers(test.mock)
	test.service.EXPECT().GetUsers().Times(3).Return(users)

	user := mock_au10.NewMockUser(test.mock)

	testToken := "test token"
	users.EXPECT().Auth("test login x").
		Return(user, &testToken, errors.New("test error"))
	test.logMembership.EXPECT().IsAvailable(test.rights).Return(false)
	test.log.EXPECT().Error(
		`345.test login: Failed to auth "test login x": "test error" (RPC error %d).`,
		codes.Internal)
	token, err := test.client.Auth(&proto.AuthRequest{Login: "test login x"})
	test.assert.Nil(token)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")

	users.EXPECT().Auth("test login x2").Return(user, nil, nil)
	test.log.EXPECT().Info(
		`345.test login: Wrong creds for "%s"`, "test login x2")
	token, err = test.client.Auth(&proto.AuthRequest{Login: "test login x2"})
	test.assert.Nil(token)
	test.assert.NoError(err)

	users.EXPECT().Auth("test login x3").Return(user, &testToken, nil)
	user.EXPECT().GetLogin().Return("test login x4")
	test.log.EXPECT().Info(`345.test login x4: Authed "%s".`, "test login x3")
	token, err = test.client.Auth(&proto.AuthRequest{Login: "test login x3"})
	test.assert.NotNil(token)
	test.assert.Equal("test token", *token)
	test.assert.NoError(err)
}

func Test_Accesspoint_Client_ReadLog(t *testing.T) {
	test := createClientTest(t)
	defer test.close()

	response := mock_proto.NewMockAu10_ReadLogServer(test.mock)

	test.testPermissionDenied(
		test.logMembership,
		func() (interface{}, error) {
			return nil, test.client.ReadLog(nil, response)
		},
		"read log")

	test.logMembership.EXPECT().IsAvailable(test.rights).Return(true)
	test.logMembership.EXPECT().IsAvailable(test.rights).Return(false)
	test.log.EXPECT().Subscribe().Return(nil, errors.New("test subscribe error"))
	test.log.EXPECT().Error(
		`345.test login: Failed to subscribe to log: "test subscribe error" (RPC error %d).`,
		codes.Internal)
	err := test.client.ReadLog(nil, response)
	test.assert.EqualError(err, "rpc error: code = Internal desc = INTERNAL")

	var recordsChan chan au10.LogRecord
	var errChan chan error

	test.testSubscription(
		func(ctx context.Context) error {
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
			test.logMembership.EXPECT().IsAvailable(test.rights).Return(true)
			response.EXPECT().Send(gomock.Any()).Do(func(record *proto.LogRecord) {
				test.assert.Equal(int64(987), record.SeqNum)
				test.assert.Equal(int64(567), record.Time)
				test.assert.Equal("test log record", record.Text)
				test.assert.Equal("debug", record.Severity)
				test.assert.Equal("test node type", record.NodeType)
				test.assert.Equal("test node name", record.NodeName)
				test.assert.Equal(6+3,
					reflect.Indirect(reflect.ValueOf(record)).NumField())
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

	test.testPermissionDenied(
		postsMembership,
		func() (interface{}, error) {
			return nil, test.client.ReadPosts(nil, response)
		},
		"read posts")
}
