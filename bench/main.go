package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"
)

var (
	//go:embed testdata/fixture_large_config.json
	test_large_config          []byte
	test_large_config_variable = "v-key-25"
)

func main() {
	configServer := newConfigServer()
	defer configServer.Close()
	log.Printf("Running stub config server at %v", configServer.URL)

	eventServer := newEventServer()
	defer eventServer.Close()
	log.Printf("Running stub event server at %v", eventServer.URL)

	client, err := devcycle.NewDVCClient(os.Getenv("DVC_SERVER_SDK_KEY"), &devcycle.DVCOptions{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         time.Second,
		ConfigPollingIntervalMS:      time.Second,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
		ConfigCDNURI:                 configServer.URL,
		EventsAPIURI:                 eventServer.URL,
	})
	if err != nil {
		log.Fatalf("Error setting up DVC client: %v", err)
	}
	dvcUser := devcycle.DVCUser{
		UserId: "dontcare",
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/variable", func(res http.ResponseWriter, req *http.Request) {
		variable, err := client.Variable(dvcUser, test_large_config_variable, false)
		if err != nil {
			log.Printf("Error calling Variable: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(res, "%v", variable.IsDefaulted)
	})
	mux.HandleFunc("/empty", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
	})
	log.Printf("Setting up http server")
	log.Print(http.ListenAndServe(":8080", mux))
}

func newConfigServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Etag", "TESTING")
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write(test_large_config)
	}))
}

func newEventServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusCreated)
		_, _ = res.Write([]byte("{}"))
	}))
}
