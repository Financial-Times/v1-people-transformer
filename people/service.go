package people

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Financial-Times/tme-reader/tmereader"
	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/pborman/uuid"
	"sync"
	"time"
)

const (
	cacheBucket = "person"
	//uppAuthority = "http://api.ft.com/system/FT-UPP"
	//tmeAuthority = "http://api.ft.com/system/FT-TME"
)

type peopleService interface {
	getPeople() ([]personLink, bool)
	getPersonByUUID(uuid string) (person, bool, error)
	isInitialised() bool
	shutdown() error
}

type peopleServiceImpl struct {
	sync.RWMutex
	repository    tmereader.Repository
	baseURL       string
	personLinks   []personLink
	taxonomyName  string
	maxTmeRecords int
	initialised   bool
	cacheFileName string
	db            *bolt.DB
}

func newPeopleService(repo tmereader.Repository, baseURL string, taxonomyName string, maxTmeRecords int, cacheFileName string) peopleService {
	s := &peopleServiceImpl{repository: repo, baseURL: baseURL, taxonomyName: taxonomyName, maxTmeRecords: maxTmeRecords, initialised: false, cacheFileName: cacheFileName}
	go func(service *peopleServiceImpl) {
		err := service.loadDB()
		if err != nil {
			log.Errorf("Error while creating PeopleService: [%v]", err.Error())
		}
	}(s)
	return s
}

func (s *peopleServiceImpl) isInitialised() bool {
	s.RLock()
	defer s.RUnlock()
	return s.initialised
}

func (s *peopleServiceImpl) shutdown() error {
	log.Info("Shutingdown...")
	if s.db == nil {
		return errors.New("DB not open")
	}
	return s.db.Close()
}

func (s *peopleServiceImpl) getPeople() ([]personLink, bool) {
	if len(s.personLinks) > 0 {
		return s.personLinks, true
	}
	return s.personLinks, false
}

func (s *peopleServiceImpl) getPersonByUUID(uuid string) (person, bool, error) {
	var cachedValue []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(cacheBucket))
		if bucket == nil {
			return fmt.Errorf("Bucket %v not found!", cacheBucket)
		}
		cachedValue = bucket.Get([]byte(uuid))
		return nil
	})

	if err != nil {
		log.Errorf("ERROR reading from cache file for [%v]: %v", uuid, err.Error())
		return person{}, false, err
	}
	if len(cachedValue) == 0 {
		log.Infof("INFO No cached value for [%v]", uuid)
		return person{}, false, nil
	}
	var cachedPerson person
	err = json.Unmarshal(cachedValue, &cachedPerson)
	if err != nil {
		log.Errorf("ERROR unmarshalling cached value for [%v]: %v", uuid, err.Error())
		return person{}, true, err
	}
	return cachedPerson, true, nil

}

func (s *peopleServiceImpl) loadDB() error {
	c := make(chan []person)
	go s.processPeople(c)
	s.Lock()
	defer func() {
		close(c)
		s.initialised = true
		s.Unlock()
	}()
	var err error
	s.db, err = bolt.Open(s.cacheFileName, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Errorf("ERROR opening cache file for init: %v", err.Error())
		return err
	}
	if err = s.createCacheBucket(); err != nil {
		return err
	}

	responseCount := 0
	for {
		terms, err := s.repository.GetTmeTermsFromIndex(responseCount)
		if err != nil {
			return err
		}
		if len(terms) < 1 {
			log.Info("Finished fetching people from TME. Waiting subroutines to terminate")
			break
		}

		log.Infof("Terms length is: %v", len(terms))
		s.processTerms(terms, c)
		responseCount += s.maxTmeRecords
	}
	return nil
}

func (s *peopleServiceImpl) processTerms(terms []interface{}, c chan []person) {
	log.Info("Processing terms...")
	var cacheToBeWritten []person
	for _, iTerm := range terms {
		t := iTerm.(term)
		tmeIdentifier := buildTmeIdentifier(t.RawID, s.taxonomyName)
		personUUID := uuid.NewMD5(uuid.UUID{}, []byte(tmeIdentifier)).String()
		s.personLinks = append(s.personLinks, personLink{APIURL: s.baseURL + "/" + personUUID})
		cacheToBeWritten = append(cacheToBeWritten, transformPerson(t, s.taxonomyName))
	}
	c <- cacheToBeWritten
}

func (s *peopleServiceImpl) processPeople(c chan []person) {
	for people := range c {
		log.Infof("Got people of: %v", people)
		err := s.db.Batch(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(cacheBucket))
			if bucket == nil {
				return fmt.Errorf("Cache bucket [%v] not found!", cacheBucket)
			}
			for _, anPerson := range people {
				marshalledPerson, err := json.Marshal(anPerson)
				if err != nil {
					return err
				}
				err = bucket.Put([]byte(anPerson.UUID), marshalledPerson)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Errorf("ERROR storing to cache: %+v", err)
		}
	}

	log.Info("Finished processing all people")
}

func (s *peopleServiceImpl) createCacheBucket() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(cacheBucket))
		if err != nil {
			log.Warnf("Cache bucket [%v] could not be deleted\n", cacheBucket)
		}
		_, err = tx.CreateBucket([]byte(cacheBucket))
		return err
	})
}
