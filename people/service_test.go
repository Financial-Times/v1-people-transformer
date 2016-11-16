package people

import (
	"github.com/Financial-Times/tme-reader/tmereader"
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

type testSuiteForPeople struct {
	name  string
	uuid  string
	found bool
	err   error
}

func createTestPeopleService(t *testing.T, repo tmereader.Repository, cacheFileName string) peopleService {
	service := newPeopleService(repo, "/base/url", "taxonomy_string", 10, cacheFileName)
	return service
}

func getTempFile(t *testing.T) *os.File {
	tmpfile, err := ioutil.TempFile("", "example")
	assert.NoError(t, err)
	assert.NoError(t, tmpfile.Close())
	log.Debug("File:%s", tmpfile.Name())
	return tmpfile
}

func waitTillInit(t *testing.T, s peopleService) {
	for i := 1; i <= 1000; i++ {
		if s.isInitialised() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestInit(t *testing.T) {
	tmpfile := getTempFile(t)
	defer os.Remove(tmpfile.Name())
	service := createTestPeopleService(t, &dummyRepo{}, tmpfile.Name())
	defer service.shutdown()
	waitTillInit(t, service)
	assert.Equal(t, true, service.isInitialised())
}

func TestGetPeople(t *testing.T) {
	tmpfile := getTempFile(t)
	defer os.Remove(tmpfile.Name())
	repo := dummyRepo{terms: []term{{CanonicalName: "Bob", RawID: "bob"}, {CanonicalName: "Fred", RawID: "fred"}}}
	service := createTestPeopleService(t, &repo, tmpfile.Name())
	defer service.shutdown()
	waitTillInit(t, service)
	peopleLinks, found := service.getPeople()
	assert.True(t, found)
	assert.Len(t, peopleLinks, 2)
}

func TestGetPersonByUUID(t *testing.T) {
	tmpfile := getTempFile(t)
	defer os.Remove(tmpfile.Name())
	repo := dummyRepo{terms: []term{{CanonicalName: "Bob", RawID: "bob"}, {CanonicalName: "Fred", RawID: "fred"}}}
	service := createTestPeopleService(t, &repo, tmpfile.Name())
	defer service.shutdown()
	waitTillInit(t, service)

	tests := []testSuiteForPeople{
		{"Success", "28d66fcc-bb56-363d-80c1-f2d957ef58cf", true, nil},
		{"Success", "xxxxxxxx-bb56-363d-80c1-f2d957ef58cf", false, nil}}
	for _, test := range tests {
		person, found, err := service.getPersonByUUID(test.uuid)
		if test.err != nil {
			assert.Equal(t, test.err, err)
		} else if test.found {
			assert.True(t, found)
			assert.NotNil(t, person)
		} else {
			assert.False(t, found)
		}
	}

}

type dummyRepo struct {
	terms []term
	err   error
}

func (d *dummyRepo) GetTmeTermsFromIndex(startRecord int) ([]interface{}, error) {
	if startRecord > 0 {
		return nil, d.err
	}
	var interfaces = make([]interface{}, len(d.terms))
	for i, data := range d.terms {
		interfaces[i] = data
	}
	return interfaces, d.err
}

func (d *dummyRepo) GetTmeTermById(uuid string) (interface{}, error) {
	return d.terms[0], d.err
}
