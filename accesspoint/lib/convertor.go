package accesspoint

import (
	fmt "fmt"
	"strconv"
	"time"

	proto "bitbucket.org/au10/service/accesspoint/proto"
	"bitbucket.org/au10/service/au10"
)

// Convertor converts protocol messages and to protocol messages.
type Convertor interface {
	PostIDFromProto(string, *error) au10.PostID
	MessageIDFromProto(string, *error) au10.MessageID
	TimeToProto(time.Time) int64
	MessageKindFromProto(proto.Message_Kind, *error) au10.MessageKind
	MessageKindToProto(au10.MessageKind, *error) proto.Message_Kind
	MessageToProto(au10.Message, *error) *proto.Message
	MessagesToProto([]au10.Message, *error) []*proto.Message
	MessageDeclarationFromProto(
		*proto.PostAddRequest_MessageDeclaration,
		*error) *au10.MessageDeclaration
	MessagesDeclarationFromProto(
		[]*proto.PostAddRequest_MessageDeclaration,
		*error) []*au10.MessageDeclaration
	GeoPointToProto(*au10.GeoPoint) *proto.GeoPoint
	GeoPointFromProto(*proto.GeoPoint) *au10.GeoPoint
	PostToProto(au10.Post, *error) *proto.Post
	VocalToProto(au10.Vocal, *error) *proto.Vocal
	VocalDeclarationFromProto(
		*proto.VocalAddRequest,
		au10.User,
		*error) *au10.VocalDeclaration
	LogRecordToProto(au10.LogRecord) *proto.LogRecord
}

// NewConvertor creates new convertor.
func NewConvertor() Convertor { return &convertor{} }

type convertor struct{}

func (convertor) idFromProto(source string, errResult *error) uint32 {
	result, err := strconv.ParseUint(source, 10, 32)
	if err != nil && *errResult == nil {
		*errResult = err
	}
	return uint32(result)
}
func (c *convertor) PostIDFromProto(source string, err *error) au10.PostID {
	return au10.PostID(c.idFromProto(source, err))
}
func (c *convertor) MessageIDFromProto(source string, err *error) au10.MessageID {
	return au10.MessageID(c.idFromProto(source, err))
}

func (convertor) idToProto(source uint32) string {
	return strconv.FormatUint(uint64(source), 10)
}
func (c *convertor) postIDToProto(source au10.PostID) string {
	return c.idToProto(uint32(source))
}
func (c *convertor) messageIDToProto(source au10.MessageID) string {
	return c.idToProto(uint32(source))
}

func (convertor) TimeToProto(source time.Time) int64 {
	return source.UnixNano()
}

func (convertor) MessageKindFromProto(
	source proto.Message_Kind, err *error) au10.MessageKind {
	switch source {
	case proto.Message_TEXT:
		return au10.MessageKindText
	}
	if *err == nil {
		*err = fmt.Errorf("unknown proto message kind %d", source)
	}
	return 0
}

func (convertor) MessageKindToProto(
	source au10.MessageKind, err *error) proto.Message_Kind {
	switch source {
	case au10.MessageKindText:
		return proto.Message_TEXT
	}
	if *err == nil {
		*err = fmt.Errorf("unknown au10 message kind %d", source)
	}
	return 0
}

func (c *convertor) MessageToProto(
	source au10.Message,
	err *error) *proto.Message {
	return &proto.Message{
		Id:   c.messageIDToProto(source.GetID()),
		Kind: c.MessageKindToProto(source.GetKind(), err),
		Size: source.GetSize()}
}

func (c *convertor) MessagesToProto(
	source []au10.Message, err *error) []*proto.Message {
	result := make([]*proto.Message, len(source))
	for i, m := range source {
		result[i] = c.MessageToProto(m, err)
	}
	return result
}

func (c *convertor) MessageDeclarationFromProto(
	source *proto.PostAddRequest_MessageDeclaration,
	err *error) *au10.MessageDeclaration {
	return &au10.MessageDeclaration{
		Kind: c.MessageKindFromProto(source.Kind, err),
		Size: source.Size}
}

func (c *convertor) MessagesDeclarationFromProto(
	source []*proto.PostAddRequest_MessageDeclaration,
	err *error) []*au10.MessageDeclaration {
	result := make([]*au10.MessageDeclaration, len(source))
	for i, m := range source {
		result[i] = c.MessageDeclarationFromProto(m, err)
		if *err != nil {
			return nil
		}
	}
	return result
}

func (convertor) GeoPointToProto(source *au10.GeoPoint) *proto.GeoPoint {
	return &proto.GeoPoint{Latitude: source.Latitude, Longitude: source.Longitude}
}
func (convertor) GeoPointFromProto(source *proto.GeoPoint) *au10.GeoPoint {
	return &au10.GeoPoint{Latitude: source.Latitude, Longitude: source.Longitude}
}

func (c *convertor) PostToProto(source au10.Post, err *error) *proto.Post {
	return &proto.Post{
		Id:       c.postIDToProto(source.GetID()),
		Time:     c.TimeToProto(source.GetTime()),
		Location: c.GeoPointToProto(source.GetLocation()),
		Messages: c.MessagesToProto(source.GetMessages(), err)}
}

func (c *convertor) VocalToProto(source au10.Vocal, err *error) *proto.Vocal {
	return &proto.Vocal{
		Post: c.PostToProto(source, err)}
}

func (c *convertor) VocalDeclarationFromProto(
	source *proto.VocalAddRequest,
	user au10.User,
	err *error) *au10.VocalDeclaration {
	return &au10.VocalDeclaration{
		PostDeclaration: au10.PostDeclaration{
			Author:   user.GetID(),
			Location: c.GeoPointFromProto(source.Post.Location),
			Messages: c.MessagesDeclarationFromProto(source.Post.Messages, err)}}
}

func (c *convertor) LogRecordToProto(source au10.LogRecord) *proto.LogRecord {
	return &proto.LogRecord{
		SeqNum:   source.GetSequenceNumber(),
		Time:     c.TimeToProto(source.GetTime()),
		Text:     source.GetText(),
		Severity: source.GetSeverity(),
		NodeType: source.GetNodeType(),
		NodeName: source.GetNodeName()}
}
