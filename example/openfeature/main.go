package main

import (
	"context"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"log"
	"os"
	"time"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"
	"github.com/open-feature/go-sdk/openfeature"
)

func main() {
	sdkKey := os.Getenv("DEVCYCLE_SERVER_SDK_KEY")
	if sdkKey == "" {
		log.Fatal("DEVCYCLE_SERVER_SDK_KEY env var not set: set it to your SDK key")
	}
	inithandler := make(chan api.ClientEvent, 10)
	dvcOptions := devcycle.Options{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         time.Second * 10,
		ConfigPollingIntervalMS:      time.Second * 10,
		RequestTimeout:               time.Second * 10,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
		ClientEventHandler:           inithandler,
	}
	dvcClient, err := devcycle.NewClient(sdkKey, &dvcOptions)
	if err != nil {
		log.Fatalf("Failed to create DevCycle client: %v", err)
	}

	// Wait for the client to be initialized
	for {
		time.Sleep(time.Millisecond * 100)
		event := <-inithandler
		if event.EventType == api.ClientEventType_Initialized {
			break
		}
	}
	if err = openfeature.SetProvider(dvcClient.OpenFeatureProvider()); err != nil {
		log.Fatalf("Failed to set DevCycle provider: %v", err)
	}
	client := openfeature.NewClient("devcycle")

	evalCtx := openfeature.NewEvaluationContext("test-1234", map[string]interface{}{
		"email":             "test-user@domain.com",
		"name":              "Test User",
		"language":          "en",
		"country":           "CA",
		"appVersion":        "1.0.0",
		"appBuild":          "1",
		"customData":        map[string]interface{}{"custom": "data"},
		"privateCustomData": map[string]interface{}{"private": "data"},
		"deviceModel":       "Macbook",
	})

	// Retrieving an object variable with a default value
	value, err := client.ObjectValue(context.Background(), "test-json-variable", map[string]interface{}{"value": "default"}, evalCtx)

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Variable results: %#v", value)

	// Checking a string variable flag
	stringVariable := "go-example-tests"
	if exampleValue, err := client.StringValue(context.Background(), stringVariable, "DEFAULT", evalCtx); err != nil {
		log.Printf("Error retrieving feature flag: %v", err)
	} else if exampleValue != "DEFAULT" {
		log.Printf("%v = %s, feature is enabled", stringVariable, exampleValue)
	}

	// Retrieving a string variable along with the resolution details
	details, err := client.StringValueDetails(context.Background(), "doesnt-exist", "DEFAULT", evalCtx)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Variable results for unknown variable: %#v", details)
}
