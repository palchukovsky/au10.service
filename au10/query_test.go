package au10_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"bitbucket.org/au10/service/postdb"

	"bitbucket.org/au10/service/au10"
	mock_postdb "bitbucket.org/au10/service/mock/postdb"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_QueryClient_GetVocal(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	db := mock_postdb.NewMockDB(mock)
	record := &postdb.Post{
		ID:                 123,
		Author:             234,
		Time:               time.Now(),
		LocaltionLongitude: 345.456,
		LocaltionLatitude:  456.789,
		Messages: []*postdb.Message{
			&postdb.Message{ID: 987, Kind: 0, Size: 654},
			&postdb.Message{ID: 543, Kind: 0, Size: 765}}}
	assert.Equal(6, reflect.Indirect(reflect.ValueOf(record)).NumField())
	assert.Equal(3,
		reflect.Indirect(reflect.ValueOf(record.Messages[0])).NumField())

	query := au10.NewQueryClient(db)
	defer query.Close()

	db.EXPECT().GetPost(uint32(667)).Return(record, errors.New("test error"))
	vocal, err := query.GetVocal(au10.PostID(667))
	assert.Nil(vocal)
	assert.EqualError(err, "test error")

	db.EXPECT().GetPost(uint32(668)).Return(record, nil)
	vocal, err = query.GetVocal(au10.PostID(668))
	assert.NoError(err)
	assert.Equal(au10.PostID(record.ID), vocal.GetID())
	assert.Equal(au10.UserID(record.Author), vocal.GetAuthor())
	assert.True(record.Time.Equal(vocal.GetTime()))
	assert.Equal(record.LocaltionLatitude, vocal.GetLocation().Latitude)
	assert.Equal(record.LocaltionLongitude, vocal.GetLocation().Longitude)
	if assert.Equal(len(record.Messages), len(vocal.GetMessages())) {
		for i, m := range vocal.GetMessages() {
			assert.Equal(au10.MessageID(record.Messages[i].ID), m.GetID())
			assert.Equal(au10.MessageKind(record.Messages[i].Kind), m.GetKind())
			assert.Equal(record.Messages[i].Size, m.GetSize())
		}
	}

	record.Messages[1].Kind = 1
	db.EXPECT().GetPost(uint32(669)).Return(record, nil)
	vocal, err = query.GetVocal(au10.PostID(669))
	assert.Nil(vocal)
	assert.EqualError(err, "unknown DB-record message kind 1")
}
