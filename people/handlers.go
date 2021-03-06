package people

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/service-status-go/gtg"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

type PeopleHandler struct {
	service PeopleService
}

func NewPeopleHandler(service PeopleService) PeopleHandler {
	return PeopleHandler{service}
}

func (h *PeopleHandler) GetPeople(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add("Content-Type", "application/json")
	if !h.service.isInitialised() {
		writeStatusServiceUnavailable(writer)
		return
	}

	if c, _ := h.service.getCount(); c == 0 {
		writeJSONMessageWithStatus(writer, "People not found", http.StatusNotFound)
		return
	}

	pv, err := h.service.getPeople()

	if err != nil {
		writeJSONMessageWithStatus(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer pv.Close()
	writer.WriteHeader(http.StatusOK)
	io.Copy(writer, &pv)
}

func (h *PeopleHandler) GetPeopleUUIDs(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add("Content-Type", "application/json")
	if !h.service.isInitialised() {
		writeStatusServiceUnavailable(writer)
		return
	}

	if c, _ := h.service.getCount(); c == 0 {
		writeJSONMessageWithStatus(writer, "People not found", http.StatusNotFound)
		return
	}

	pv, err := h.service.getPeopleUUIDs()

	if err != nil {
		writeJSONMessageWithStatus(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer pv.Close()
	writer.WriteHeader(http.StatusOK)
	io.Copy(writer, &pv)
}

func (h *PeopleHandler) GetCount(writer http.ResponseWriter, req *http.Request) {
	if !h.service.isInitialised() {
		writer.Header().Add("Content-Type", "application/json")
		writeStatusServiceUnavailable(writer)
		return
	}
	count, err := h.service.getCount()
	if err != nil {
		writer.Header().Add("Content-Type", "application/json")
		writeJSONMessageWithStatus(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Write([]byte(strconv.Itoa(count)))
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
	if h.service.isInitialised() && err == nil && count > 0 {
		return gtg.Status{GoodToGo: true}
	}
	return gtg.Status{GoodToGo: false}
}

func (h *PeopleHandler) GetPersonByUUID(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add("Content-Type", "application/json")
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

func (h *PeopleHandler) Reload(writer http.ResponseWriter, req *http.Request) {
	if !h.service.isInitialised() || !h.service.isDataLoaded() {
		writeStatusServiceUnavailable(writer)
		return
	}

	go func() {
		if err := h.service.reloadDB(); err != nil {
			log.Errorf("ERROR opening db: %v", err.Error())
		}
	}()
	writeJSONMessageWithStatus(writer, "Reloading people", http.StatusAccepted)
}

func writeJSONResponse(obj interface{}, found bool, writer http.ResponseWriter) {
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

//OnlyPostAllowed - Used to tell the user the METHOD type is not POST.
func (h *PeopleHandler) OnlyPostAllowed(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Allow", "POST")
	writer.WriteHeader(http.StatusMethodNotAllowed)
}

//OnlyGetAllowed - Used to tell the user the METHOD type is not GET.
func (h *PeopleHandler) OnlyGetAllowed(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Allow", "GET")
	writer.WriteHeader(http.StatusMethodNotAllowed)
}
