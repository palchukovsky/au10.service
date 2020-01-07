package accesspoint

import (
	fmt "fmt"
	"strconv"
	"time"

	proto "bitbucket.org/au10/service/accesspoint/proto"
	"bitbucket.org/au10/service/au10"
)

func convertIDFromProto(source string, errResult *error) uint64 {
	result, err := strconv.ParseUint(source, 10, 64)
	if err != nil && *errResult == nil {
		*errResult = err
	}
	return result
}
func convertIDToProto(source uint64) string {
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
		Id:   convertIDToProto(uint64(source.GetID())),
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

func convertGeoPointToProto(source *au10.GeoPoint) *proto.GeoPoint {
	return &proto.GeoPoint{Latitude: source.Latitude, Longitude: source.Longitude}
}

func convertPostToProto(source au10.Post, err *error) *proto.Post {
	return &proto.Post{
		Id:       convertIDToProto(uint64(source.GetID())),
		Time:     convertTimeToProto(source.GetTime()),
		Location: convertGeoPointToProto(source.GetLocation()),
		Messages: convertMessagesToProto(source.GetMessages(), err)}
}

func convertVocalToProto(source au10.Vocal, err *error) *proto.Vocal {
	return &proto.Vocal{
		Post: convertPostToProto(source, err)}
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
