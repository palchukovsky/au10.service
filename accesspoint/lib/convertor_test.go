package accesspoint_test

import (
	"reflect"
	"testing"
	"time"

	ap "bitbucket.org/au10/service/accesspoint/lib"
	proto "bitbucket.org/au10/service/accesspoint/proto"
	"bitbucket.org/au10/service/au10"
	mock_au10 "bitbucket.org/au10/service/mock/au10"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func testNumberOfStructFields(
	assert *assert.Assertions,
	s interface{},
	numberOfFields int) {
	assert.Equal(numberOfFields,
		reflect.Indirect(reflect.ValueOf(s)).NumField())
}

func testNumberOfProtoStructFields(
	assert *assert.Assertions,
	s interface{},
	numberOfFields int) {
	testNumberOfStructFields(assert, s, numberOfFields+3)
}

func Test_Accesspoint_Convertor_IDsFromProto(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	convertor := ap.NewConvertor()
	var err error

	assert.Equal(au10.PostID(123), convertor.PostIDFromProto("123", &err))
	assert.NoError(err)
	assert.Equal(au10.PostID(0), convertor.PostIDFromProto("ghj", &err))
	assert.EqualError(err, `strconv.ParseUint: parsing "ghj": invalid syntax`)

	err = nil
	assert.Equal(au10.MessageID(123), convertor.MessageIDFromProto("123", &err))
	assert.NoError(err)
	assert.Equal(au10.MessageID(0), convertor.MessageIDFromProto("ghj", &err))
	assert.EqualError(err, `strconv.ParseUint: parsing "ghj": invalid syntax`)
}

func Test_Accesspoint_Convertor_VocalToProto_Success(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	source := mock_au10.NewMockVocal(mock)
	source.EXPECT().GetID().Return(au10.PostID(987))
	source.EXPECT().GetTime().Return(time.Unix(0, 567))
	source.EXPECT().GetLocation().Return(
		&au10.GeoPoint{Latitude: 123.456, Longitude: 456.789})
	message1 := mock_au10.NewMockMessage(mock)
	message1.EXPECT().GetID().Return(au10.MessageID(890))
	message1.EXPECT().GetKind().Return(au10.MessageKindText)
	message1.EXPECT().GetSize().Return(uint32(564))
	message2 := mock_au10.NewMockMessage(mock)
	message2.EXPECT().GetID().Return(au10.MessageID(892))
	message2.EXPECT().GetKind().Return(au10.MessageKindText)
	message2.EXPECT().GetSize().Return(uint32(565))
	source.EXPECT().GetMessages().Return([]au10.Message{message1, message2})

	var err error
	result := ap.NewConvertor().VocalToProto(source, &err)
	assert.NoError(err)

	assert.NotNil(result)
	assert.Equal("987", result.Post.Id)
	assert.Equal(int64(567), result.Post.Time)
	assert.Equal(123.456, result.Post.Location.Latitude)
	assert.Equal(456.789, result.Post.Location.Longitude)
	testNumberOfProtoStructFields(assert, result.Post.Location, 2)
	assert.Equal([]*proto.Message{
		&proto.Message{Id: "890", Kind: proto.Message_TEXT, Size: uint32(564)},
		&proto.Message{Id: "892", Kind: proto.Message_TEXT, Size: uint32(565)}},
		result.Post.Messages)
	assert.Equal(1, len(proto.Message_Kind_value))
	testNumberOfProtoStructFields(assert, result.Post.Messages[0], 3)
	testNumberOfProtoStructFields(assert, result.Post, 4)
	testNumberOfProtoStructFields(assert, result, 1)
}
func Test_Accesspoint_Convertor_VocalToProto_Error(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	source := mock_au10.NewMockVocal(mock)
	source.EXPECT().GetID().Return(au10.PostID(987))
	source.EXPECT().GetTime().Return(time.Unix(0, 567))
	source.EXPECT().GetLocation().Return(
		&au10.GeoPoint{Latitude: 123.456, Longitude: 456.789})
	message1 := mock_au10.NewMockMessage(mock)
	message1.EXPECT().GetID().Return(au10.MessageID(890))
	message1.EXPECT().GetKind().Return(au10.MessageKindText)
	message1.EXPECT().GetSize().Return(uint32(564))
	message2 := mock_au10.NewMockMessage(mock)
	message2.EXPECT().GetID().Return(au10.MessageID(892))
	message2.EXPECT().GetKind().Return(au10.MessageKind(1))
	message2.EXPECT().GetSize().Return(uint32(565))
	source.EXPECT().GetMessages().Return([]au10.Message{message1, message2})

	var err error
	ap.NewConvertor().VocalToProto(source, &err)
	assert.EqualError(err, "unknown au10 message kind 1")
}

func Test_Accesspoint_Convertor_VocalDeclarationFromProto_Success(
	test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	source := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{
			Location: &proto.GeoPoint{Latitude: 123.456, Longitude: 456.789},
			Messages: []*proto.PostAddRequest_MessageDeclaration{
				&proto.PostAddRequest_MessageDeclaration{
					Kind: proto.Message_TEXT,
					Size: 123},
				&proto.PostAddRequest_MessageDeclaration{
					Kind: proto.Message_TEXT,
					Size: 456}}}}
	assert.Equal(1, len(proto.Message_Kind_name))
	testNumberOfProtoStructFields(assert, source, 1)
	testNumberOfProtoStructFields(assert, source.Post, 2)
	testNumberOfProtoStructFields(assert, source.Post.Messages[0], 2)

	user := mock_au10.NewMockUser(mock)
	user.EXPECT().GetID().Return(au10.UserID(567))

	var err error
	result := ap.NewConvertor().VocalDeclarationFromProto(source, user, &err)
	assert.NoError(err)

	assert.NotNil(result)
	assert.Equal(au10.UserID(567), result.Author)
	assert.Equal(123.456, result.Location.Latitude)
	assert.Equal(456.789, result.Location.Longitude)
	testNumberOfStructFields(assert, result.Location, 2)
	assert.Equal(
		[]*au10.MessageDeclaration{
			&au10.MessageDeclaration{Kind: au10.MessageKindText, Size: uint32(123)},
			&au10.MessageDeclaration{Kind: au10.MessageKindText, Size: uint32(456)}},
		result.Messages)
	assert.Equal(1, len(proto.Message_Kind_value))
	testNumberOfStructFields(assert, result.Messages[0], 2)
	testNumberOfStructFields(assert, result.PostDeclaration, 3)
	testNumberOfStructFields(assert, result, 1)
}

func Test_Accesspoint_Convertor_VocalDeclarationFromProto_Error(
	test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	source := &proto.VocalAddRequest{
		Post: &proto.PostAddRequest{
			Location: &proto.GeoPoint{Latitude: 123.456, Longitude: 456.789},
			Messages: []*proto.PostAddRequest_MessageDeclaration{
				&proto.PostAddRequest_MessageDeclaration{
					Kind: proto.Message_TEXT,
					Size: 123},
				&proto.PostAddRequest_MessageDeclaration{
					Kind: 1,
					Size: 456}}}}
	assert.Equal(1, len(proto.Message_Kind_name))
	testNumberOfProtoStructFields(assert, source, 1)
	testNumberOfProtoStructFields(assert, source.Post, 2)
	testNumberOfProtoStructFields(assert, source.Post.Messages[0], 2)

	user := mock_au10.NewMockUser(mock)
	user.EXPECT().GetID().Return(au10.UserID(567))

	var err error
	ap.NewConvertor().VocalDeclarationFromProto(source, user, &err)
	assert.EqualError(err, "unknown proto message kind 1")
}

func Test_Accesspoint_Convertor_LogRecordToProto(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	source := mock_au10.NewMockLogRecord(mock)
	source.EXPECT().GetSequenceNumber().Return(int64(987))
	source.EXPECT().GetTime().Return(time.Unix(0, 567))
	source.EXPECT().GetText().Return("test log record")
	source.EXPECT().GetSeverity().Return("debug")
	source.EXPECT().GetNodeType().Return("test node type")
	source.EXPECT().GetNodeName().Return("test node name")

	result := ap.NewConvertor().LogRecordToProto(source)

	assert.NotNil(result)
	assert.Equal(int64(987), result.SeqNum)
	assert.Equal(int64(567), result.Time)
	assert.Equal("test log record", result.Text)
	assert.Equal("debug", result.Severity)
	assert.Equal("test node type", result.NodeType)
	assert.Equal("test node name", result.NodeName)
	testNumberOfProtoStructFields(assert, result, 6)
}
