package main

import (
	"encoding/json"
	"expvar"
	"flag"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/HdrHistogram/hdrhistogram-go"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"

	"time"
)

type EventMetrics struct {
	enableTracking   bool
	numEventPayloads *expvar.Int
	numBadPayloads   *expvar.Int
	numEventBatches  *expvar.Int
	numEvents        *expvar.Int
}

func main() {
	var enableEvents bool
	var configInterval, eventFlushInterval time.Duration
	var flushEventQueueSize int
	var maxEventQueueSize int
	var maxMemoryBuckets int
	var maxWASMWorkers int
	var listenAddr string
	var numUniqueVariables int
	var configFailureChance int
	var eventFailureChance int
	var enableFullProfiling bool
	var enableDatadog bool
	var datadogEnv string
	var disableLogging bool
	var trackEventMetrics bool

	flag.BoolVar(&enableEvents, "enable-events", true, "enable event logging")
	flag.DurationVar(&configInterval, "config-interval", time.Minute, "interval between checks for config updates")
	flag.DurationVar(&eventFlushInterval, "event-interval", time.Minute, "interval between flushing events")
	flag.IntVar(&maxMemoryBuckets, "max-memory-buckets", 0, "set max memory allocation buckets")
	flag.IntVar(&maxWASMWorkers, "max-wasm-workers", 0, "set number of WASM workers (zero defaults to GOMAXPROCS)")
	flag.StringVar(&listenAddr, "listen", ":8080", "[host]:port to listen on")
	flag.IntVar(&numUniqueVariables, "num-variables", 85, "Unique variables to use in multipleVariables endpoint")
	flag.IntVar(&configFailureChance, "config-failure-chance", 0, "Chance of config server returning 500")
	flag.IntVar(&eventFailureChance, "event-failure-chance", 0, "Chance of event server returning 500")
	flag.IntVar(&flushEventQueueSize, "flush-event-queue-size", 5000, "Max events to hold before flushing")
	flag.IntVar(&maxEventQueueSize, "max-event-queue-size", 50000, "Max events to hold before dropping")
	flag.BoolVar(&enableFullProfiling, "enable-full-profiling", false, "Enable full profiling, at the cost of performance")
	flag.BoolVar(&enableDatadog, "datadog", true, "Use datadog for tracing and profiling")
	flag.StringVar(&datadogEnv, "datadog-env", "benchmark", "Datadog environment to use")
	flag.BoolVar(&disableLogging, "disable-logging", false, "Turns off detailed logging in the SDK")
	flag.BoolVar(&trackEventMetrics, "track-event-metrics", true, "Enables processing and tracking of variable event metrics, this may cause some performance issues")

	flag.Parse()

	if enableDatadog {
		tracer.Start(
			tracer.WithService("go-bench-dd"),
			tracer.WithEnv(datadogEnv),
			tracer.WithRuntimeMetrics(),
			tracer.WithGlobalTag("num-variables", strconv.Itoa(numUniqueVariables)),
			tracer.WithGlobalTag("max-wasm-workers", strconv.Itoa(maxWASMWorkers)),
			tracer.WithGlobalTag("max-memory-buckets", strconv.Itoa(maxMemoryBuckets)),
			tracer.WithGlobalTag("enable-events", strconv.FormatBool(enableEvents)),
			tracer.WithServiceVersion(devcycle.VERSION),
		)

		defer tracer.Stop()

		profileTypes := profiler.WithProfileTypes(
			profiler.CPUProfile,
			profiler.HeapProfile,
		)

		if enableFullProfiling {
			profileTypes = profiler.WithProfileTypes(
				profiler.CPUProfile,
				profiler.HeapProfile,
				profiler.BlockProfile,
				profiler.MutexProfile,
				profiler.GoroutineProfile,
			)
		}

		err := profiler.Start(
			profiler.WithService("go-bench-dd"),
			profiler.WithEnv("benchmark"),
			profileTypes,
		)
		if err != nil {
			log.Fatal(err)
		}
		defer profiler.Stop()
	}

	configServer := newConfigServer(configFailureChance)
	defer configServer.Close()
	log.Printf("Running stub config server at %v", configServer.URL)

	var eventMetrics = EventMetrics{
		enableTracking:   trackEventMetrics,
		numEventPayloads: expvar.NewInt("event_payloads_total"),
		numBadPayloads:   expvar.NewInt("bad_payloads_total"),
		numEventBatches:  expvar.NewInt("event_batches_total"),
		numEvents:        expvar.NewInt("events_total"),
	}

	eventServer := newEventServer(eventFailureChance, &eventMetrics)
	defer eventServer.Close()
	log.Printf("Running stub event server at %v", eventServer.URL)

	if disableLogging {
		log.Printf("Running with logging disabled")
		log.SetOutput(io.Discard)
	} else {
		log.Printf("Running with logging enabled")
	}

	client, err := devcycle.NewDVCClient("dvc_server_hello", &devcycle.DVCOptions{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         eventFlushInterval,
		ConfigPollingIntervalMS:      configInterval,
		DisableAutomaticEventLogging: !enableEvents,
		DisableCustomEventLogging:    !enableEvents,
		ConfigCDNURI:                 configServer.URL,
		EventsAPIURI:                 eventServer.URL,
		FlushEventQueueSize:          flushEventQueueSize,
		MaxEventQueueSize:            maxEventQueueSize,
		AdvancedOptions: devcycle.AdvancedOptions{
			MaxMemoryAllocationBuckets: maxMemoryBuckets,
			MaxWasmWorkers:             maxWASMWorkers,
		},
	})

	if err != nil {
		log.Fatalf("Error setting up DVC client: %v", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Error getting hostname: %v", err)
	}

	err = client.SetClientCustomData(map[string]interface{}{
		"environment": "load_test",
		"hostname":    hostname,
	})

	if err != nil {
		log.Fatalf("Error setting client custom data: %v", err)
	}

	// export goroutines at /debug/vars
	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))

	numVariableCalls := expvar.NewInt("variable_calls_total")
	sumVariableNanos := expvar.NewInt("variable_call_duration_nanos_total")

	// Lowest discernable value is 1 nanosecond, highest is 1 second, with 3 significant figures
	histogram := hdrhistogram.New(1, 1e9, 3)
	histogramLock := sync.Mutex{}

	// export percentile nanoseconds per variable evaluation at /debug/vars
	for _, percentile := range []int{50, 90, 99, 100} {
		value := float64(percentile)
		expvar.Publish(fmt.Sprintf("nanos_per_variable_%d", percentile), expvar.Func(func() interface{} {
			return histogram.ValueAtPercentile(value)
		}))
		expvar.Publish(fmt.Sprintf("duration_per_variable_%d", percentile), expvar.Func(func() interface{} {
			return time.Duration(histogram.ValueAtPercentile(value)).String()
		}))
	}

	mux := httptrace.NewServeMux()

	var userPool = setupUserPool()
	mux.HandleFunc("/variable", func(res http.ResponseWriter, req *http.Request) {
		i := rand.Intn(len(userPool))
		start := time.Now()
		variable, err := client.Variable(userPool[i], test_large_config_variable, false)
		if err != nil {
			log.Printf("Error calling Variable: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		numVariableCalls.Add(1)

		duration := time.Since(start).Nanoseconds()
		sumVariableNanos.Add(duration)

		histogramLock.Lock()
		err = histogram.RecordValue(duration)
		histogramLock.Unlock()
		if err != nil {
			fmt.Errorf("Error recording histogram value: %v", err)
			res.WriteHeader(500)
		}

		fmt.Fprintf(res, "%v\n", variable.Value)
	})

	mux.HandleFunc("/multipleVariables", func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()
		variable := devcycle.Variable{}

		for j := 0; j < numUniqueVariables; j++ {
			i := rand.Intn(len(userPool))
			result, err := client.Variable(userPool[i], fmt.Sprintf("var-%d", j), false)
			variable = result
			if err != nil {
				log.Printf("Error calling Variables: %v", err)
				res.WriteHeader(http.StatusInternalServerError)
				return
			}
			numVariableCalls.Add(1)
		}

		duration := time.Since(start).Nanoseconds()
		sumVariableNanos.Add(duration)

		histogramLock.Lock()

		err = histogram.RecordValue(duration)
		histogramLock.Unlock()
		if err != nil {
			fmt.Errorf("Error recording histogram value: %v", err)
		}
		fmt.Fprintf(res, "%v\n", variable.Value)

	})

	mux.HandleFunc("/empty", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
	})

	// Add explicit routes for expvar and pprof
	mux.Handle("/debug/vars", expvar.Handler())
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	log.Printf("HTTP server listening on %s", listenAddr)
	log.Printf("Access pprof data on http://localhost:%s/debug/pprof/", strings.Split(listenAddr, ":")[1])
	log.Printf("Access expvar data on http://localhost:%s/debug/vars", strings.Split(listenAddr, ":")[1])
	log.Print(http.ListenAndServe(listenAddr, mux))
}

func newConfigServer(failureChance int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if rand.Intn(100) < failureChance {
			res.WriteHeader(http.StatusInternalServerError)
			_, _ = res.Write([]byte("{}"))
			return
		}
		res.Header().Set("Etag", "TESTING")
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write(test_large_config)
	}))
}

func newEventServer(failureChance int, metrics *EventMetrics) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if metrics.enableTracking {
			metrics.numEventPayloads.Add(1)
			body, err := io.ReadAll(req.Body)
			if err != nil {
				metrics.numBadPayloads.Add(1)
			} else {
				var data api.BatchEventsBody
				err := json.Unmarshal(body, &data)
				if err != nil {
					metrics.numBadPayloads.Add(1)
				} else {
					metrics.numEventBatches.Add(int64(len(data.Batch)))
					for _, batch := range data.Batch {
						metrics.numEvents.Add(int64(len(batch.Events)))
					}
				}
			}
			_ = req.Body.Close()
		}

		if rand.Intn(100) < failureChance {
			res.WriteHeader(http.StatusInternalServerError)
			_, _ = res.Write([]byte("{}"))
			return
		}
		res.WriteHeader(http.StatusCreated)
		_, _ = res.Write([]byte("{}"))
	}))
}
