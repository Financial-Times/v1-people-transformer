package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type testSuiteForOrgs struct {
	name    string
	baseURL string
	terms   []term
	orgs    []personLink
	found   bool
	err     error
}

func TestGetPeople(t *testing.T) {
	peopleServiceImpl := peopleServiceImpl{initialised: true}
	assert.Equal(t, true, peopleServiceImpl.initialised)
}
