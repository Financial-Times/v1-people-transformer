package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Financial-Times/go-fthealth/v1a"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

type peopleHandler struct {
	service peopleService
}

func newPeopleHandler(service peopleService) peopleHandler {
	return peopleHandler{service: service}
}

func (peopleHandler *peopleHandler) checker() (string, error) {
	//TODO: Implement further
	err := errors.New("TODO")
	if err == nil {
		return "Connectivity to downstream systems is ok", err
	}
	return "Error connecting to downstream systems", err
}

func (peopleHandler *peopleHandler) HealthCheck() v1a.Check {
	return v1a.Check{
		BusinessImpact:   "Unable to respond",
		Name:             "Check connectivity to downstream systems",
		PanicGuide:       "TODO complete",
		Severity:         1,
		TechnicalSummary: "TODO complete",
		Checker:          peopleHandler.checker,
	}
}

func (h *peopleHandler) GoodToGo(writer http.ResponseWriter, req *http.Request) {
	// TODO: Implement
}

func (h *peopleHandler) count(writer http.ResponseWriter, req *http.Request) {
	// TODO: Implement
}

func (h *peopleHandler) getPeopleUuids(writer http.ResponseWriter, req *http.Request) {
	// TODO: Implement
}

func (h *peopleHandler) getPeople(writer http.ResponseWriter, req *http.Request) {
	if !h.service.isInitialised() {
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	obj, found := h.service.getPeople()
	writeJSONResponse(obj, found, writer)
}

func (h *peopleHandler) getPersonByUUID(writer http.ResponseWriter, req *http.Request) {
	if !h.service.isInitialised() {
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(req)
	uuid := vars["uuid"]

	obj, found, err := h.service.getPersonByUUID(uuid)
	if err != nil {
		writeJSONError(writer, err.Error(), http.StatusInternalServerError)
	}
	writeJSONResponse(obj, found, writer)
}

func writeJSONResponse(obj interface{}, found bool, writer http.ResponseWriter) {
	writer.Header().Add("Content-Type", "application/json")

	if !found {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(writer)
	if err := enc.Encode(obj); err != nil {
		log.Errorf("Error on json encoding=%v\n", err)
		writeJSONError(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

func writeJSONError(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.WriteHeader(statusCode)
	fmt.Fprintln(w, fmt.Sprintf("{\"message\": \"%s\"}", errorMsg))
}
