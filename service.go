package main

import (
	"github.com/Financial-Times/tme-reader/tmereader"
	"github.com/boltdb/bolt"
)

const (
	cacheBucket  = "person"
	uppAuthority = "http://api.ft.com/system/FT-UPP"
	tmeAuthority = "http://api.ft.com/system/FT-TME"
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
	orgLinks      []personLink
	taxonomyName  string
	maxTmeRecords int
	initialised   bool
	cacheFileName string
	db            *bolt.DB
}

func newPeopleService(repo tmereader.Repository, baseURL string, taxonomyName string, maxTmeRecords int, cacheFileName string) peopleService {
	s := &peopleServiceImpl{repository: repo, baseURL: baseURL, taxonomyName: taxonomyName, maxTmeRecords: maxTmeRecords, initialised: false, cacheFileName: cacheFileName}
	return s
}

func (s *peopleServiceImpl) isInitialised() bool {
	return s.initialised
}

// TODO: Implement
func (s *peopleServiceImpl) shutdown() error {
	return nil
}

// TODO: Implement
func (s *peopleServiceImpl) init() error {
	return nil
}

// TODO: Implement
func (s *peopleServiceImpl) getPeople() ([]personLink, bool) {
	return nil, false
}

// TODO: Implement
func (s *peopleServiceImpl) getPersonByUUID(uuid string) (person, bool, error) {
	return person{}, false, nil
}
