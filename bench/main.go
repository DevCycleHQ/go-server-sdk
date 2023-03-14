package main

import (
	_ "embed"
	"fmt"
	devcycle "github.com/devcyclehq/go-server-sdk/v2"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	_ "net/http/pprof"

	"time"
)

var (
	//go:embed testdata/fixture_large_config.json
	test_large_config          []byte
	test_large_config_variable = "v-key-25"
	lettersAndNumbers          = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = lettersAndNumbers[rand.Intn(len(lettersAndNumbers))]
	}
	return string(b)
}

func setupUserPool() []devcycle.DVCUser {
	users := make([]devcycle.DVCUser, 10)
	for i := 0; i < 10; i++ {
		customData := map[string]interface{}{
			"cacheKey":   randSeq(250),
			"propInt":    rand.Intn(1000),
			"propDouble": rand.Float32(),
			"propBool":   rand.Intn(2) == 1,
		}

		customPrivateData := map[string]interface{}{
			"aPrivateValue": "secret-data-here",
		}

		users[i] = devcycle.DVCUser{
			UserId:            fmt.Sprintf("user_%d", i),
			DeviceModel:       "testing",
			Name:              fmt.Sprintf("Testing User %d", i),
			Email:             fmt.Sprintf("test.user-%d@gmail.com", i),
			AppBuild:          "1.0.0",
			AppVersion:        "0.0.1",
			Country:           "ca",
			Language:          "en",
			CustomData:        customData,
			PrivateCustomData: customPrivateData,
		}
	}
	return users
}

func main() {
	configServer := newConfigServer()
	defer configServer.Close()
	log.Printf("Running stub config server at %v", configServer.URL)

	eventServer := newEventServer()
	defer eventServer.Close()
	log.Printf("Running stub event server at %v", eventServer.URL)

	client, err := devcycle.NewDVCClient("dvc_server_hello", &devcycle.DVCOptions{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         time.Second,
		ConfigPollingIntervalMS:      time.Second,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
		ConfigCDNURI:                 configServer.URL,
		EventsAPIURI:                 eventServer.URL,
		UseDebugWasm:                 true,
	})
	if err != nil {
		log.Fatalf("Error setting up DVC client: %v", err)
	}

	var userPool = setupUserPool()
	mux := http.NewServeMux()
	mux.HandleFunc("/variable", func(res http.ResponseWriter, req *http.Request) {
		i := rand.Intn(len(userPool))
		variable, err := client.Variable(userPool[i], test_large_config_variable, false)
		if err != nil {
			log.Printf("Error calling Variable: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(res, "%v", variable)
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
