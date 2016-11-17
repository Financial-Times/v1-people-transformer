package people

import (
	"fmt"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/service-status-go/gtg"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

const (
	testUUID                = "bba39990-c78d-3629-ae83-808c333c6dbc"
	getPeopleResponse       = "[{\"apiUrl\":\"http://localhost:8080/transformers/people/bba39990-c78d-3629-ae83-808c333c6dbc\"}]\n"
	getPersonByUUIDResponse = "{\"uuid\":\"bba39990-c78d-3629-ae83-808c333c6dbc\",\"prefLabel\":\"European Union\",\"type\":\"Organisation\",\"alternativeIdentifiers\":{\"TME\":[\"MTE3-U3ViamVjdHM=\"],\"uuids\":[\"bba39990-c78d-3629-ae83-808c333c6dbc\"]}}\n"
)

func TestHandlers(t *testing.T) {
	tests := []struct {
		name         string
		req          *http.Request
		dummyService PeopleService
		statusCode   int
		contentType  string // Contents of the Content-Type header
		body         string
	}{
		{"Success - get person by uuid", newRequest("GET", fmt.Sprintf("/transformers/people/%s", testUUID)), &dummyService{found: true, initialised: true, people: []person{{UUID: testUUID, PrefLabel: "European Union", AlternativeIdentifiers: alternativeIdentifiers{Uuids: []string{testUUID}, TME: []string{"MTE3-U3ViamVjdHM="}}, Type: "Organisation"}}}, http.StatusOK, "application/json", getPersonByUUIDResponse},
		{"Not found - get person by uuid", newRequest("GET", fmt.Sprintf("/transformers/people/%s", testUUID)), &dummyService{found: false, initialised: true, people: []person{{}}}, http.StatusNotFound, "application/json", "{\"message\": \"Person not found\"}\n"},
		{"Service unavailable - get person by uuid", newRequest("GET", fmt.Sprintf("/transformers/people/%s", testUUID)), &dummyService{found: false, initialised: false, people: []person{}}, http.StatusServiceUnavailable, "application/json", "{\"message\": \"Service Unavailable\"}\n"},
		{"Success - get people", newRequest("GET", "/transformers/people"), &dummyService{found: true, initialised: true, people: []person{{UUID: testUUID}}}, http.StatusOK, "application/json", getPeopleResponse},
		{"Success - get people count", newRequest("GET", "/transformers/people/__count"), &dummyService{found: true, count: 1, initialised: true, people: []person{{UUID: testUUID}}}, http.StatusOK, "application/json", "1\n"},
		{"Not found - get people", newRequest("GET", "/transformers/people"), &dummyService{found: false, initialised: true, people: []person{}}, http.StatusNotFound, "application/json", "{\"message\": \"Person not found\"}\n"},
		{"Service unavailable - get people", newRequest("GET", "/transformers/people"), &dummyService{found: false, initialised: false, people: []person{}}, http.StatusServiceUnavailable, "application/json", "{\"message\": \"Service Unavailable\"}\n"},
		{"GTG unavailable - get GTG", newRequest("GET", status.GTGPath), &dummyService{found: false, initialised: false, people: []person{}}, http.StatusServiceUnavailable, "application/json", ""},
		{"GTG unavailable - get GTG but no people", newRequest("GET", status.GTGPath), &dummyService{found: false, initialised: true, people: []person{}}, http.StatusServiceUnavailable, "application/json", ""},
		{"GTG OK - get GTG", newRequest("GET", status.GTGPath), &dummyService{found: true, initialised: true, count: 2, people: []person{}}, http.StatusOK, "application/json", "OK"},
		{"Health bad - get Health check", newRequest("GET", "/__health"), &dummyService{found: false, initialised: false}, http.StatusOK, "application/json", "regex=Service is initilising"},
		{"Health good - get Health check", newRequest("GET", "/__health"), &dummyService{found: false, initialised: true}, http.StatusOK, "application/json", "regex=Service is up and running"},
	}
	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(test.dummyService).ServeHTTP(rec, test.req)
		assert.Equal(t, test.statusCode, rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))

		if strings.HasPrefix(test.body, "regex=") {
			regex := strings.TrimPrefix(test.body, "regex=")
			body := rec.Body.String()
			matched, err := regexp.MatchString(regex, body)
			assert.NoError(t, err)
			assert.True(t, matched, fmt.Sprintf("Could not match regex:\n %s \nin body:\n %s", regex, body))
		} else {
			assert.Equal(t, test.body, rec.Body.String(), fmt.Sprintf("%s: Wrong body", test.name))
		}
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
	count       int
	err         error
}

func (s *dummyService) getPeople() ([]personLink, bool) {
	var links []personLink
	for _, sub := range s.people {
		links = append(links, personLink{APIURL: "http://localhost:8080/transformers/people/" + sub.UUID})
	}
	return links, s.found
}

func (s *dummyService) getCount() (int, error) {
	return s.count, s.err
}

func (s *dummyService) getPersonByUUID(uuid string) (person, bool, error) {
	return s.people[0], s.found, nil
}

func (s *dummyService) isInitialised() bool {
	return s.initialised
}

func (s *dummyService) Shutdown() error {
	return s.err
}

func router(s PeopleService) *mux.Router {
	m := mux.NewRouter()
	h := NewPeopleHandler(s)
	m.HandleFunc("/transformers/people", h.GetPeople).Methods("GET")
	m.HandleFunc("/transformers/people/__count", h.GetCount).Methods("GET")
	m.HandleFunc("/transformers/people/{uuid}", h.GetPersonByUUID).Methods("GET")
	m.HandleFunc("/__health", v1a.Handler("V1 People Transformer Healthchecks", "Checks for the health of the service", h.HealthCheck()))
	g2gHandler := status.NewGoodToGoHandler(gtg.StatusChecker(h.G2GCheck))
	m.HandleFunc(status.GTGPath, g2gHandler)
	return m
}
