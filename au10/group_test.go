package au10_test

import (
	"testing"

	"bitbucket.org/au10/service/au10"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_Rights_Creation(test *testing.T) {
	assert := assert.New(test)
	rights := au10.NewRights("test domain", "test name")
	assert.Equal("test domain", rights.Get().Domain)
	assert.Equal("test name", rights.Get().Name)

	rights = au10.NewRights("*", "test name")
	assert.Equal("*", rights.Get().Domain)
	assert.Equal("test name", rights.Get().Name)

	rights = au10.NewRights("test domain", "*")
	assert.Equal("test domain", rights.Get().Domain)
	assert.Equal("*", rights.Get().Name)
}

func Test_Au10_CheckRights_Root(test *testing.T) {
	assert := assert.New(test)
	rights := []au10.Rights{
		au10.NewRights("test domain", "test group"),
		au10.NewRights("*", "*")}
	assert.True(au10.NewMembership("some domain", "some group").IsAllowed(rights))
	assert.True(au10.NewMembership("test domain", "test group").IsAllowed(rights))
	assert.True(au10.NewMembership("", "some group").IsAllowed(rights))
	assert.True(au10.NewMembership("some domain", "").IsAllowed(rights))
	assert.True(au10.NewMembership("", "").IsAllowed(rights))
	assert.True(au10.NewMembership("*", "some group").IsAllowed(rights))
	assert.True(au10.NewMembership("some domain", "*").IsAllowed(rights))
	assert.True(au10.NewMembership("*", "*").IsAllowed(rights))
}

func Test_Au10_CheckRights_DomainAdmin(test *testing.T) {
	assert := assert.New(test)
	rights := []au10.Rights{
		au10.NewRights("test domain 1", "test group"),
		au10.NewRights("test domain 2", "test group"),
		au10.NewRights("test domain 2", "*")}
	assert.True(!au10.NewMembership("some domain", "test group").IsAllowed(rights))
	assert.True(au10.NewMembership("test domain 2", "test group").IsAllowed(rights))
	assert.True(au10.NewMembership("test domain 2", "test group X").IsAllowed(rights))
	assert.True(!au10.NewMembership("", "").IsAllowed(rights))
	assert.True(au10.NewMembership("test domain 2", "").IsAllowed(rights))
	assert.True(!au10.NewMembership("test domain 1", "").IsAllowed(rights))
	assert.True(au10.NewMembership("*", "test group").IsAllowed(rights))
	assert.True(au10.NewMembership("*", "test group X").IsAllowed(rights))
	assert.True(au10.NewMembership("*", "").IsAllowed(rights))
	assert.True(au10.NewMembership("*", "*").IsAllowed(rights))
	assert.True(au10.NewMembership("test domain 1", "*").IsAllowed(rights))
	assert.True(au10.NewMembership("test domain 2", "*").IsAllowed(rights))
	assert.True(!au10.NewMembership("test domain 3", "*").IsAllowed(rights))
}

func Test_Au10_CheckRights_User(test *testing.T) {
	assert := assert.New(test)
	rights := []au10.Rights{
		au10.NewRights("test domain 1", "test group"),
		au10.NewRights("test domain 2", "test group 1"),
		au10.NewRights("test domain 2", "test group 2")}
	assert.True(!au10.NewMembership("some domain", "test group").IsAllowed(rights))
	assert.True(au10.NewMembership("test domain 1", "test group").IsAllowed(rights))
	assert.True(!au10.NewMembership("test domain 1", "test group X").IsAllowed(rights))
	assert.True(au10.NewMembership("test domain 2", "test group 1").IsAllowed(rights))
	assert.True(au10.NewMembership("test domain 2", "test group 2").IsAllowed(rights))
	assert.True(!au10.NewMembership("test domain 2", "test group X").IsAllowed(rights))
	assert.True(au10.NewMembership("*", "test group 1").IsAllowed(rights))
	assert.True(!au10.NewMembership("*", "test group X").IsAllowed(rights))
	assert.True(au10.NewMembership("test domain 1", "*").IsAllowed(rights))
	assert.True(au10.NewMembership("*", "*").IsAllowed(rights))
	assert.True(!au10.NewMembership("", "test group 1").IsAllowed(rights))
	assert.True(!au10.NewMembership("", "").IsAllowed(rights))
	assert.True(!au10.NewMembership("test domain 1", "").IsAllowed(rights))
}

func Test_Au10_CheckRights_NoRights(test *testing.T) {
	assert := assert.New(test)
	rights := []au10.Rights{}
	assert.True(!au10.NewMembership("some domain", "some group").IsAllowed(rights))
	assert.True(!au10.NewMembership("*", "test group 1").IsAllowed(rights))
	assert.True(!au10.NewMembership("*", "test group 2").IsAllowed(rights))
	assert.True(!au10.NewMembership("*", "test group X").IsAllowed(rights))
	assert.True(!au10.NewMembership("test domain", "*").IsAllowed(rights))
	assert.True(!au10.NewMembership("test domain X", "*").IsAllowed(rights))
	assert.True(au10.NewMembership("*", "*").IsAllowed(rights))
}

func Test_Au10_Membership_Creation(test *testing.T) {
	assert := assert.New(test)
	membership := au10.NewMembership("test domain", "test name")
	assert.Equal("test domain", membership.Get().Domain)
	assert.Equal("test name", membership.Get().Name)

	membership = au10.NewMembership("*", "test name")
	assert.Equal("*", membership.Get().Domain)
	assert.Equal("test name", membership.Get().Name)

	membership = au10.NewMembership("test domain", "*")
	assert.Equal("test domain", membership.Get().Domain)
	assert.Equal("*", membership.Get().Name)
}
