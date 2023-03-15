package main

import (
	_ "embed"
	"expvar"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	_ "net/http/pprof"
	"runtime"
	"strings"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"

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
	var enableEvents bool
	var configInterval, eventFlushInterval time.Duration
	var maxMemoryBuckets int
	var maxWASMWorkers int
	var listenAddr string

	flag.BoolVar(&enableEvents, "enable-events", false, "enable event logging")
	flag.DurationVar(&configInterval, "config-interval", time.Minute, "interval between checks for config updates")
	flag.DurationVar(&eventFlushInterval, "event-interval", time.Minute, "interval between flushing events")
	flag.IntVar(&maxMemoryBuckets, "max-memory-buckets", 0, "set max memory allocation buckets")
	flag.IntVar(&maxWASMWorkers, "max-wasm-workers", 0, "set number of WASM workers (zero defaults to GOMAXPROCS)")
	flag.StringVar(&listenAddr, "listen", ":8080", "[host]:port to listen on")
	flag.Parse()

	configServer := newConfigServer()
	defer configServer.Close()
	log.Printf("Running stub config server at %v", configServer.URL)

	eventServer := newEventServer()
	defer eventServer.Close()
	log.Printf("Running stub event server at %v", eventServer.URL)

	client, err := devcycle.NewDVCClient("dvc_server_hello", &devcycle.DVCOptions{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         eventFlushInterval,
		ConfigPollingIntervalMS:      configInterval,
		DisableAutomaticEventLogging: !enableEvents,
		DisableCustomEventLogging:    !enableEvents,
		ConfigCDNURI:                 configServer.URL,
		EventsAPIURI:                 eventServer.URL,
		AdvancedOptions: devcycle.AdvancedOptions{
			MaxMemoryAllocationBuckets: maxMemoryBuckets,
			MaxWasmWorkers:             maxWASMWorkers,
		},
	})

	if err != nil {
		log.Fatalf("Error setting up DVC client: %v", err)
	}

	// export goroutines at /debug/vars
	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))

	var userPool = setupUserPool()
	http.HandleFunc("/variable", func(res http.ResponseWriter, req *http.Request) {
		i := rand.Intn(len(userPool))
		variable, err := client.Variable(userPool[i], test_large_config_variable, false)
		if err != nil {
			log.Printf("Error calling Variable: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(res, "%v\n", variable.Value)
	})
	http.HandleFunc("/empty", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
	})
	log.Printf("HTTP server listening on %s", listenAddr)
	log.Printf("Access pprof data on http://localhost:%s/debug/pprof/", strings.Split(listenAddr, ":")[1])
	log.Printf("Access expvar data on http://localhost:%s/debug/vars", strings.Split(listenAddr, ":")[1])
	log.Print(http.ListenAndServe(":8080", nil))
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
