package people

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/service-status-go/gtg"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

type PeopleHandler struct {
	service PeopleService
}

func NewPeopleHandler(service PeopleService) PeopleHandler {
	return PeopleHandler{service}
}

func (h *PeopleHandler) GetPeople(writer http.ResponseWriter, req *http.Request) {
	if !h.service.isInitialised() {
		writeStatusServiceUnavailable(writer)
		return
	}
	obj, found := h.service.getPeople()
	writeJSONResponse(obj, found, writer)
}

func (h *PeopleHandler) GetCount(writer http.ResponseWriter, req *http.Request) {
	if !h.service.isInitialised() {
		writeStatusServiceUnavailable(writer)
		return
	}
	count, err := h.service.getCount()
	if err != nil {
		writeJSONMessageWithStatus(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(count, true, writer)
}

func (h *PeopleHandler) HealthCheck() v1a.Check {

	return v1a.Check{
		BusinessImpact:   "Unable to respond to requests",
		Name:             "Check service has finished initilising.",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/v1-people-transformer",
		Severity:         1,
		TechnicalSummary: "Cannot serve any content as data not loaded.",
		Checker: func() (string, error) {
			if h.service.isInitialised() {
				return "Service is up and running", nil
			}
			return "Error as service initilising", errors.New("Service is initilising.")
		},
	}
}

func (h *PeopleHandler) G2GCheck() gtg.Status {
	count, err := h.service.getCount()
	if err == nil && count > 0 {
		return gtg.Status{GoodToGo: true}
	}
	return gtg.Status{GoodToGo: false}
}

func (h *PeopleHandler) GetPersonByUUID(writer http.ResponseWriter, req *http.Request) {
	if !h.service.isInitialised() {
		writeStatusServiceUnavailable(writer)
		return
	}

	vars := mux.Vars(req)
	uuid := vars["uuid"]

	obj, found, err := h.service.getPersonByUUID(uuid)
	if err != nil {
		writeJSONMessageWithStatus(writer, err.Error(), http.StatusInternalServerError)
	}
	writeJSONResponse(obj, found, writer)
}

func writeJSONResponse(obj interface{}, found bool, writer http.ResponseWriter) {
	writer.Header().Add("Content-Type", "application/json")

	if !found {
		writeJSONMessageWithStatus(writer, "Person not found", http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(writer)
	if err := enc.Encode(obj); err != nil {
		log.Errorf("Error on json encoding=%v", err)
		writeJSONMessageWithStatus(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

func writeJSONMessageWithStatus(w http.ResponseWriter, msg string, statusCode int) {
	w.WriteHeader(statusCode)
	fmt.Fprintln(w, fmt.Sprintf("{\"message\": \"%s\"}", msg))
}

func writeStatusServiceUnavailable(w http.ResponseWriter) {
	writeJSONMessageWithStatus(w, "Service Unavailable", http.StatusServiceUnavailable)
}
