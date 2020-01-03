package au10_test

import (
	"testing"

	"bitbucket.org/au10/service/au10"
	"github.com/stretchr/testify/assert"
)

func Test_Au10_Rights_Creation(test *testing.T) {
	assert := assert.New(test)
	rights := au10.CreateRights("test domain", "test name")
	assert.Equal("test domain", rights.Get().Domain)
	assert.Equal("test name", rights.Get().Name)

	rights = au10.CreateRights("*", "test name")
	assert.Equal("*", rights.Get().Domain)
	assert.Equal("test name", rights.Get().Name)

	rights = au10.CreateRights("test domain", "*")
	assert.Equal("test domain", rights.Get().Domain)
	assert.Equal("*", rights.Get().Name)
}

func Test_Au10_CheckRights_Root(test *testing.T) {
	assert := assert.New(test)
	rights := []au10.Rights{
		au10.CreateRights("test domain", "test group"),
		au10.CreateRights("*", "*")}
	assert.True(au10.CreateMembership("some domain", "some group").IsAllowed(rights))
	assert.True(au10.CreateMembership("test domain", "test group").IsAllowed(rights))
	assert.True(au10.CreateMembership("", "some group").IsAllowed(rights))
	assert.True(au10.CreateMembership("some domain", "").IsAllowed(rights))
	assert.True(au10.CreateMembership("", "").IsAllowed(rights))
	assert.True(au10.CreateMembership("*", "some group").IsAllowed(rights))
	assert.True(au10.CreateMembership("some domain", "*").IsAllowed(rights))
	assert.True(au10.CreateMembership("*", "*").IsAllowed(rights))
}

func Test_Au10_CheckRights_DomainAdmin(test *testing.T) {
	assert := assert.New(test)
	rights := []au10.Rights{
		au10.CreateRights("test domain 1", "test group"),
		au10.CreateRights("test domain 2", "test group"),
		au10.CreateRights("test domain 2", "*")}
	assert.True(!au10.CreateMembership("some domain", "test group").IsAllowed(rights))
	assert.True(au10.CreateMembership("test domain 2", "test group").IsAllowed(rights))
	assert.True(au10.CreateMembership("test domain 2", "test group X").IsAllowed(rights))
	assert.True(!au10.CreateMembership("", "").IsAllowed(rights))
	assert.True(au10.CreateMembership("test domain 2", "").IsAllowed(rights))
	assert.True(!au10.CreateMembership("test domain 1", "").IsAllowed(rights))
	assert.True(au10.CreateMembership("*", "test group").IsAllowed(rights))
	assert.True(au10.CreateMembership("*", "test group X").IsAllowed(rights))
	assert.True(au10.CreateMembership("*", "").IsAllowed(rights))
	assert.True(au10.CreateMembership("*", "*").IsAllowed(rights))
	assert.True(au10.CreateMembership("test domain 1", "*").IsAllowed(rights))
	assert.True(au10.CreateMembership("test domain 2", "*").IsAllowed(rights))
	assert.True(!au10.CreateMembership("test domain 3", "*").IsAllowed(rights))
}

func Test_Au10_CheckRights_User(test *testing.T) {
	assert := assert.New(test)
	rights := []au10.Rights{
		au10.CreateRights("test domain 1", "test group"),
		au10.CreateRights("test domain 2", "test group 1"),
		au10.CreateRights("test domain 2", "test group 2")}
	assert.True(!au10.CreateMembership("some domain", "test group").IsAllowed(rights))
	assert.True(au10.CreateMembership("test domain 1", "test group").IsAllowed(rights))
	assert.True(!au10.CreateMembership("test domain 1", "test group X").IsAllowed(rights))
	assert.True(au10.CreateMembership("test domain 2", "test group 1").IsAllowed(rights))
	assert.True(au10.CreateMembership("test domain 2", "test group 2").IsAllowed(rights))
	assert.True(!au10.CreateMembership("test domain 2", "test group X").IsAllowed(rights))
	assert.True(au10.CreateMembership("*", "test group 1").IsAllowed(rights))
	assert.True(!au10.CreateMembership("*", "test group X").IsAllowed(rights))
	assert.True(au10.CreateMembership("test domain 1", "*").IsAllowed(rights))
	assert.True(au10.CreateMembership("*", "*").IsAllowed(rights))
	assert.True(!au10.CreateMembership("", "test group 1").IsAllowed(rights))
	assert.True(!au10.CreateMembership("", "").IsAllowed(rights))
	assert.True(!au10.CreateMembership("test domain 1", "").IsAllowed(rights))
}

func Test_Au10_CheckRights_NoRights(test *testing.T) {
	assert := assert.New(test)
	rights := []au10.Rights{}
	assert.True(!au10.CreateMembership("some domain", "some group").IsAllowed(rights))
	assert.True(!au10.CreateMembership("*", "test group 1").IsAllowed(rights))
	assert.True(!au10.CreateMembership("*", "test group 2").IsAllowed(rights))
	assert.True(!au10.CreateMembership("*", "test group X").IsAllowed(rights))
	assert.True(!au10.CreateMembership("test domain", "*").IsAllowed(rights))
	assert.True(!au10.CreateMembership("test domain X", "*").IsAllowed(rights))
	assert.True(au10.CreateMembership("*", "*").IsAllowed(rights))
}

func Test_Au10_Membership_Creation(test *testing.T) {
	assert := assert.New(test)
	membership := au10.CreateMembership("test domain", "test name")
	assert.Equal("test domain", membership.Get().Domain)
	assert.Equal("test name", membership.Get().Name)

	membership = au10.CreateMembership("*", "test name")
	assert.Equal("*", membership.Get().Domain)
	assert.Equal("test name", membership.Get().Name)

	membership = au10.CreateMembership("test domain", "*")
	assert.Equal("test domain", membership.Get().Domain)
	assert.Equal("*", membership.Get().Name)
}
