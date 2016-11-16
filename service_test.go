package main

import "testing"

type testSuiteForOrgs struct {
	name    string
	baseURL string
	terms   []term
	orgs    []personLink
	found   bool
	err     error
}

func TestGetPeople(t *testing.T) {
	// TODO: Implement
}
