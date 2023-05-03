package main

import (
	"context"
	devcycle "github.com/devcyclehq/go-server-sdk/v2"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"log"
	"os"
	"time"
)

func main() {
	sdkKey := os.Getenv("DVC_SERVER_KEY")
	if sdkKey == "" {
		log.Fatal("DVC_SERVER_KEY env var not set: set it to your SDK key")
	}

	dvcOptions := devcycle.Options{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         true,
		EventFlushIntervalMS:         time.Second * 10,
		ConfigPollingIntervalMS:      time.Second * 10,
		RequestTimeout:               time.Second * 10,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
	}
	dvcClient, _ := devcycle.NewClient(sdkKey, &dvcOptions)
	openfeature.SetProvider(devcycle.DevCycleProvider{Client: dvcClient})
	client := openfeature.NewClient("hello")

	evalCtx := openfeature.NewEvaluationContext("n/a", map[string]interface{}{
		"userId":            "123",
		"email":             "chris.hoefgen@taplytics.com",
		"name":              "Chris Hoefgen",
		"language":          "en",
		"country":           "CA",
		"appVersion":        "1.0.0",
		"appBuild":          "1",
		"customData":        map[string]interface{}{"custom": "data"},
		"privateCustomData": map[string]interface{}{"private": "data"},
		"deviceModel":       "Macbook",
	})

	value, err := client.ObjectValue(context.Background(), "json-testing", map[string]interface{}{"default": "value"}, evalCtx)

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Variable results: %v | %T \n", value, value)

	details, err := client.BooleanValueDetails(context.Background(), "doesnt-exist", false, evalCtx)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Variable results: %v\n", details)
}
