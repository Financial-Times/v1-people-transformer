package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testUUID = "d907cd7d-dfd0-4800-8d2b-229911439a96"
const getPeopleResponse = "[{\"apiUrl\":\"http://localhost:8080/transformers/people/bba39990-c78d-3629-ae83-808c333c6dbc\"}]\n"
const getPersonByUUIDResponse = "{\"uuid\":\"d907cd7d-dfd0-4800-8d2b-229911439a96\",\"prefLabel\":\"Joe Bloggs\",\"type\":\"Person\",\"alternativeIdentifiers\":{" +
	"\"TME\":[\"MTE3-U3ViamVjdHM=\"]," +
	"\"uuids\":[\"d907cd7d-dfd0-4800-8d2b-229911439a96c\"]" +
	"}}\n"

func TestHandlers(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		name         string
		req          *http.Request
		dummyService peopleService
		statusCode   int
		body         string
	}{
		{"Success - get person by uuid",
			newRequest("GET", fmt.Sprintf("/transformers/people/%s", testUUID)),
			&dummyService{
				found:       true,
				initialised: true,
				people: []person{person{
					UUID:      testUUID,
					PrefLabel: "Joe Bloggs",
					AlternativeIdentifiers: alternativeIdentifiers{
						Uuids: []string{testUUID}, TME: []string{"MTE3-U3ViamVjdHM="}},
					Type: "Person"}}},
			http.StatusOK,
			getPersonByUUIDResponse},
		{"Not found - get person by uuid",
			newRequest("GET", fmt.Sprintf("/transformers/people/%s", testUUID)),
			&dummyService{
				found:       false,
				initialised: true,
				people:      []person{}},
			http.StatusNotFound,
			""},
		{"Service unavailable - get person by uuid",
			newRequest("GET", fmt.Sprintf("/transformers/people/%s", testUUID)),
			&dummyService{
				found:       false,
				initialised: false,
				people:      []person{}},
			http.StatusServiceUnavailable,
			""},
		{"Success - get people",
			newRequest("GET", "/transformers/people"),
			&dummyService{found: true, initialised: true, people: []person{person{UUID: testUUID}}},
			http.StatusOK,
			getPeopleResponse},
		{"Not found - get people",
			newRequest("GET", "/transformers/people"),
			&dummyService{found: false, initialised: true, people: []person{}},
			http.StatusNotFound,
			""},
		{"Service unavailable - get people",
			newRequest("GET", "/transformers/people"),
			&dummyService{found: false, initialised: false, people: []person{}},
			http.StatusServiceUnavailable,
			""},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(peopleHandler{test.dummyService}).ServeHTTP(rec, test.req)
		assert.True(test.statusCode == rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.Equal(test.body, rec.Body.String(), fmt.Sprintf("%s: Wrong body", test.name))
	}
}

func newRequest(method, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

type dummyService struct {
	found       bool
	people      []person
	initialised bool
}

func (s *dummyService) getPeople() ([]personLink, bool) {
	var peopleLinks []personLink
	for _, sub := range s.people {
		peopleLinks = append(peopleLinks, personLink{APIURL: "http://localhost:8080/transformers/people/" + sub.UUID})
	}
	return peopleLinks, s.found
}

func (s *dummyService) getPersonByUUID(uuid string) (person, bool, error) {
	person := person{UUID: uuid}
	return person, s.found, nil
}

func (s *dummyService) isInitialised() bool {
	return s.initialised
}

func (s *dummyService) shutdown() error {
	return nil
}
