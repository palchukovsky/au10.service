package au10_test

import (
	"testing"
	"time"

	"bitbucket.org/au10/service/au10"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_Factory(test *testing.T) {
	assert := assert.New(test)

	factory := au10.NewFactory()

	assert.Equal(3*time.Second, factory.NewRedialSleepTime())
}
