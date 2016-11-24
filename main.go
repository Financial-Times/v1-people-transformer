package main

import (
	"fmt"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/service-status-go/gtg"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/Financial-Times/tme-reader/tmereader"
	"github.com/Financial-Times/v1-people-transformer/people"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/sethgrid/pester"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

func main() {
	app := cli.App("v1-people-transformer", "A RESTful API for transforming TME People to UP json")
	username := app.String(cli.StringOpt{
		Name:   "tme-username",
		Value:  "",
		Desc:   "TME username used for http basic authentication",
		EnvVar: "TME_USERNAME",
	})
	password := app.String(cli.StringOpt{
		Name:   "tme-password",
		Value:  "",
		Desc:   "TME password used for http basic authentication",
		EnvVar: "TME_PASSWORD",
	})
	token := app.String(cli.StringOpt{
		Name:   "token",
		Value:  "",
		Desc:   "Token to be used for accessig TME",
		EnvVar: "TOKEN",
	})
	baseURL := app.String(cli.StringOpt{
		Name:   "base-url",
		Value:  "http://localhost:8080/transformers/people/",
		Desc:   "Base url",
		EnvVar: "BASE_URL",
	})
	tmeBaseURL := app.String(cli.StringOpt{
		Name:   "tme-base-url",
		Value:  "https://tme.ft.com",
		Desc:   "TME base url",
		EnvVar: "TME_BASE_URL",
	})
	port := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		Desc:   "Port to listen on",
		EnvVar: "PORT",
	})
	maxRecords := app.Int(cli.IntOpt{
		Name:   "maxRecords",
		Value:  int(10000),
		Desc:   "Maximum records to be queried to TME",
		EnvVar: "MAX_RECORDS",
	})
	batchSize := app.Int(cli.IntOpt{
		Name:   "batchSize",
		Value:  int(10),
		Desc:   "Number of requests to be executed in parallel to TME",
		EnvVar: "BATCH_SIZE",
	})
	cacheFileName := app.String(cli.StringOpt{
		Name:   "cache-file-name",
		Value:  "cache.db",
		Desc:   "Cache file name",
		EnvVar: "CACHE_FILE_NAME",
	})

	tmeTaxonomyName := "PN"

	app.Action = func() {
		client := getResilientClient()
		modelTransformer := new(people.PersonTransformer)
		s := people.NewPeopleService(
			tmereader.NewTmeRepository(
				client,
				*tmeBaseURL,
				*username,
				*password,
				*token,
				*maxRecords,
				*batchSize,
				tmeTaxonomyName,
				&tmereader.AuthorityFiles{},
				modelTransformer),
			*baseURL,
			tmeTaxonomyName,
			*maxRecords,
			*cacheFileName)
		defer s.Shutdown()
		handler := people.NewPeopleHandler(s)
		r := router(handler)
		http.Handle("/", r)

		log.Printf("listening on %d", *port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
		if err != nil {
			log.Errorf("Error by listen and serve: %v", err.Error())
		}
	}
	app.Run(os.Args)
}

func router(handler people.PeopleHandler) http.Handler {
	servicesRouter := mux.NewRouter()
	// The top one of these feels more correct, but the lower one matches what we have in Dropwizard,
	// so it's what apps expect currently same as ping
	servicesRouter.HandleFunc(status.PingPath, status.PingHandler)
	servicesRouter.HandleFunc(status.PingPathDW, status.PingHandler)
	servicesRouter.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)
	servicesRouter.HandleFunc(status.BuildInfoPathDW, status.BuildInfoHandler)

	servicesRouter.HandleFunc("/__health", v1a.Handler("V1 People Transformer Healthchecks", "Checks for the health of the service", handler.HealthCheck()))

	g2gHandler := status.NewGoodToGoHandler(gtg.StatusChecker(handler.G2GCheck))
	servicesRouter.HandleFunc(status.GTGPath, g2gHandler)

	servicesRouter.HandleFunc("/transformers/people/{uuid:([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})}", handler.GetPersonByUUID).Methods("GET")
	servicesRouter.HandleFunc("/transformers/people", handler.GetPeople).Methods("GET")
	servicesRouter.HandleFunc("/transformers/people/__count", handler.GetCount).Methods("GET")
	servicesRouter.HandleFunc("/transformers/people/__ids", handler.GetPeopleUUIDs).Methods("GET")
	servicesRouter.HandleFunc("/transformers/people/__reload", handler.Reload).Methods("POST")

	return servicesRouter
}

func getResilientClient() *pester.Client {
	tr := &http.Transport{
		MaxIdleConnsPerHost: 32,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
	}
	c := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
	client := pester.NewExtendedClient(c)
	client.Backoff = pester.ExponentialBackoff
	client.MaxRetries = 5
	client.Concurrency = 1

	return client
}
