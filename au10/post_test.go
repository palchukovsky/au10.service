package au10_test

import (
	"reflect"
	"testing"
	"time"

	"bitbucket.org/au10/service/au10"
	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_Post(test *testing.T) {
	assert := assert.New(test)

	post, err := au10.ConvertSaramaMessageIntoPost(&sarama.ConsumerMessage{
		Key: []byte("test key"),
		Value: []byte(
			`{
					"i": 567,
					"t": 678,
					"l": {"x": 987.765, "y": 543.123},
					"m": [
						{"i": 34, "k": 0, "s": 987},
						{"i": 789, "k": 0, "s": 9874}
					]
				}`)})
	assert.NoError(err)
	if !assert.NotNil(post) {
		return
	}
	assert.Equal(4,
		reflect.Indirect(reflect.ValueOf(&au10.PostData{})).NumField())

	assert.Equal(au10.PostID(567), post.GetID())
	assert.Equal(time.Unix(0, 678), post.GetTime())
	assert.Equal(au10.GeoPoint{Latitude: 987.765, Longitude: 543.123},
		*post.GetLocation())

	messages := post.GetMessages()
	assert.Equal(2, len(messages))
	assert.Equal(au10.MessageID(34), messages[0].GetID())
	assert.Equal(au10.MessageID(789), messages[1].GetID())
	assert.Equal(au10.MessageKindText, messages[0].GetKind())
	assert.Equal(au10.MessageKindText, messages[1].GetKind())
	assert.Equal(uint64(987), messages[0].GetSize())
	assert.Equal(uint64(9874), messages[1].GetSize())
	assert.Equal(3,
		reflect.Indirect(reflect.ValueOf(&au10.MessageData{})).NumField())

	assert.Equal(au10.NewMembership("", ""), post.GetMembership())

	post, err = au10.ConvertSaramaMessageIntoPost(&sarama.ConsumerMessage{
		Key:   []byte("test key"),
		Value: []byte(`{`)})
	assert.Nil(post)
	assert.NotNil(err)
}
