package main

import (
	"log"
	"os"
	"time"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"
)

func main() {
	sdkKey := os.Getenv("DEVCYCLE_SERVER_SDK_KEY")
	if sdkKey == "" {
		log.Fatal("DEVCYCLE_SERVER_SDK_KEY env var not set: set it to your SDK key")
	}
	variableKey := os.Getenv("DEVCYCLE_VARIABLE_KEY")
	if variableKey == "" {
		log.Fatal("DEVCYCLE_VARIABLE_KEY env var not set: set it to a variable key")
	}

	user := devcycle.User{UserId: "test", CustomData: map[string]interface{}{
		"number-custom-property":     1334,
		"number-custom-property-3dp": 1334.610,
		"number-custom-property-2dp": 1334.61,
		"number-custom-property-1dp": 1334.6,
		"number-custom-property-0dp": 1334,
	}}

	dvcOptions := devcycle.Options{
		EventFlushIntervalMS:    0,
		ConfigPollingIntervalMS: 10 * time.Second,
		RequestTimeout:          10 * time.Second,
	}

	client, err := devcycle.NewClient(sdkKey, &dvcOptions)
	time.Sleep(10 * time.Second)
	if err != nil {
		log.Fatalf("Error initializing client: %v", err)
	}
	log.Printf("client initialized")

	variable, err := client.Variable(user, "go-custom-data", false)
	variable2, err2 := client.Variable(user, "go-custom-data", false)
	if err != nil {
		log.Fatalf("Error getting variable %v: %v", "go-custom-data", err)
	}

	if err2 != nil {
		log.Fatalf("Error getting variable %v: %v", "go-custom-data", err2)
	}
	log.Printf("variable %v: value=%v (%v) defaulted=%t", variable.Key, variable.Value, variable.Type_, variable.IsDefaulted)
	log.Printf("variable %v: value=%v (%v) defaulted=%t", variable2.Key, variable2.Value, variable2.Type_, variable2.IsDefaulted)

	return

	features, _ := client.AllFeatures(user)
	for key, feature := range features {
		log.Printf("features Key:%s, feature:%#v", key, feature)
	}

	variables, _ := client.AllVariables(user)
	for key, variable := range variables {
		log.Printf("variables Key:%s, variable:%#v", key, variable)
	}

	existingVariable, err := client.Variable(user, variableKey, "DEFAULT")
	if err != nil {
		log.Fatalf("Error getting variable %v: %v", variableKey, err)
	}
	log.Printf("variable %v: value=%v (%v) defaulted=%t", existingVariable.Key, existingVariable.Value, existingVariable.Type_, existingVariable.IsDefaulted)
	if existingVariable.IsDefaulted {
		log.Printf("Warning: variable %v should not be defaulted", existingVariable.Key)
	}

	variableValue, err := client.VariableValue(user, variableKey, "DEFAULT")
	if err != nil {
		log.Fatalf("Error getting variable value %v: %v", variableValue, err)
	}
	log.Printf("variable value=%v", variableValue)

	missingVariable, _ := client.Variable(user, variableKey+"-does-not-exist", "DEFAULT")
	if err != nil {
		log.Fatalf("Error getting variable: %v", err)
	}
	log.Printf("variable %v: value=%v (%v) defaulted=%t", missingVariable.Key, missingVariable.Value, missingVariable.Type_, missingVariable.IsDefaulted)
	if !missingVariable.IsDefaulted {
		log.Printf("Warning: variable %v should be defaulted", missingVariable.Key)
	}

	event := devcycle.Event{
		Type_:  "customEvent",
		Target: "somevariable.key",
	}
	_, err = client.Track(user, event)
	if err != nil {
		log.Fatalf("Error tracking event: %v", err)
	}

	err = client.FlushEvents()
	if err != nil {
		log.Fatalf("Error flushing events: %v", err)
	}

	time.Sleep(2 * time.Second)

	client.Close()

}
