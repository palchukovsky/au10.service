package au10_test

import (
	"testing"

	"bitbucket.org/au10/service/au10"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_PostData(test *testing.T) {
	assert := assert.New(test)

	data := au10.CreatePostData()
	assert.Equal("", data.GetText())

	data.SetText("test text")
	assert.Equal("test text", data.GetText())

	data.SetText("")
	assert.Equal("", data.GetText())
}
