package accesspoint

import (
	fmt "fmt"
	"strconv"
	"time"

	proto "bitbucket.org/au10/service/accesspoint/proto"
	"bitbucket.org/au10/service/au10"
)

func convertIDFromProto(source string, errResult *error) uint32 {
	result, err := strconv.ParseUint(source, 10, 32)
	if err != nil && *errResult == nil {
		*errResult = err
	}
	return uint32(result)
}
func convertIDToProto(source uint32) string {
	return strconv.FormatUint(uint64(source), 10)
}

func convertTimeToProto(source time.Time) int64 { return source.UnixNano() }

func convertMessageKindFromProto(
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

func convertMessageKindToProto(
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

func convertMessageToProto(source au10.Message, err *error) *proto.Message {
	return &proto.Message{
		Id:   convertIDToProto(uint32(source.GetID())),
		Kind: convertMessageKindToProto(source.GetKind(), err),
		Size: source.GetSize()}
}

func convertMessagesToProto(
	source []au10.Message, err *error) []*proto.Message {
	result := make([]*proto.Message, len(source))
	for i, m := range source {
		result[i] = convertMessageToProto(m, err)
	}
	return result
}

func convertMessageDeclarationFromProto(
	source *proto.PostAddRequest_MessageDeclaration,
	err *error) *au10.MessageDeclaration {
	return &au10.MessageDeclaration{
		Kind: convertMessageKindFromProto(source.Kind, err),
		Size: source.Size}
}

func convertMessagesDeclarationFromProto(
	source []*proto.PostAddRequest_MessageDeclaration,
	err *error) []*au10.MessageDeclaration {
	result := make([]*au10.MessageDeclaration, len(source))
	for i, m := range source {
		result[i] = convertMessageDeclarationFromProto(m, err)
		if *err != nil {
			return nil
		}
	}
	return result
}

func convertGeoPointToProto(source *au10.GeoPoint) *proto.GeoPoint {
	return &proto.GeoPoint{Latitude: source.Latitude, Longitude: source.Longitude}
}
func convertGeoPointFromProto(source *proto.GeoPoint) *au10.GeoPoint {
	return &au10.GeoPoint{Latitude: source.Latitude, Longitude: source.Longitude}
}

func convertPostToProto(source au10.Post, err *error) *proto.Post {
	return &proto.Post{
		Id:       convertIDToProto(uint32(source.GetID())),
		Time:     convertTimeToProto(source.GetTime()),
		Location: convertGeoPointToProto(source.GetLocation()),
		Messages: convertMessagesToProto(source.GetMessages(), err)}
}

func convertVocalToProto(source au10.Vocal, err *error) *proto.Vocal {
	return &proto.Vocal{
		Post: convertPostToProto(source, err)}
}

func covertVocalDeclarationFromProto(
	source *proto.VocalAddRequest,
	user au10.User,
	err *error) *au10.VocalDeclaration {
	return &au10.VocalDeclaration{
		PostDeclaration: au10.PostDeclaration{
			Author:   user.GetID(),
			Location: convertGeoPointFromProto(source.Post.Location),
			Messages: convertMessagesDeclarationFromProto(source.Post.Messages, err)}}
}

func convertLogRecordToProto(source au10.LogRecord) *proto.LogRecord {
	return &proto.LogRecord{
		SeqNum:   source.GetSequenceNumber(),
		Time:     convertTimeToProto(source.GetTime()),
		Text:     source.GetText(),
		Severity: source.GetSeverity(),
		NodeType: source.GetNodeType(),
		NodeName: source.GetNodeName()}
}
