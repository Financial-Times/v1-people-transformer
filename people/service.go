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
		err := service.init()
		if err != nil {
			log.Errorf("Error while creating OrgService: [%v]", err.Error())
		}
		service.initialised = true
	}(s)
	return s
}

func (s *peopleServiceImpl) isInitialised() bool {
	return s.initialised
}

func (s *peopleServiceImpl) shutdown() error {
	log.Info("Shutingdowwn...")
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
	var cachedOrg person
	err = json.Unmarshal(cachedValue, &cachedOrg)
	if err != nil {
		log.Errorf("ERROR unmarshalling cached value for [%v]: %v", uuid, err.Error())
		return person{}, true, err
	}
	return cachedOrg, true, nil

}

func (s *peopleServiceImpl) init() error {
	var err error
	s.db, err = bolt.Open(s.cacheFileName, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Errorf("ERROR opening cache file for init: %v", err.Error())
		return err
	}
	if err = createCacheBucket(s.db); err != nil {
		return err
	}
	var wg sync.WaitGroup
	responseCount := 0
	log.Printf("Fetching people from TME\n")
	for {
		terms, err := s.repository.GetTmeTermsFromIndex(responseCount)
		if err != nil {
			return err
		}
		if len(terms) < 1 {
			log.Printf("Finished fetching people from TME. Waiting subroutines to terminate\n")
			break
		}
		wg.Add(1)
		go s.initOrgsMap(terms, s.db, &wg)
		responseCount += s.maxTmeRecords
	}
	wg.Wait()
	log.Printf("Added %d person links\n", len(s.personLinks))
	return nil
}

func createCacheBucket(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(cacheBucket))
		if err != nil {
			log.Warnf("Cache bucket [%v] could not be deleted\n", cacheBucket)
		}
		_, err = tx.CreateBucket([]byte(cacheBucket))
		return err
	})
}

func (s *peopleServiceImpl) initOrgsMap(terms []interface{}, db *bolt.DB, wg *sync.WaitGroup) {
	var cacheToBeWritten []person
	for _, iTerm := range terms {
		t := iTerm.(term)
		tmeIdentifier := buildTmeIdentifier(t.RawID, s.taxonomyName)
		personUUID := uuid.NewMD5(uuid.UUID{}, []byte(tmeIdentifier)).String()
		s.personLinks = append(s.personLinks, personLink{APIURL: s.baseURL + "/" + personUUID})
		cacheToBeWritten = append(cacheToBeWritten, transformPerson(t, s.taxonomyName))
	}

	go storePersonToCache(db, cacheToBeWritten, wg)
}

func storePersonToCache(db *bolt.DB, cacheToBeWritten []person, wg *sync.WaitGroup) {
	defer wg.Done()
	err := db.Batch(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte(cacheBucket))
		if bucket == nil {
			return fmt.Errorf("Cache bucket [%v] not found!", cacheBucket)
		}
		for _, anPerson := range cacheToBeWritten {
			marshalledOrg, err := json.Marshal(anPerson)
			if err != nil {
				return err
			}
			err = bucket.Put([]byte(anPerson.UUID), marshalledOrg)
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
