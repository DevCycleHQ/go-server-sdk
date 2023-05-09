package main

import (
	"log"
	"os"
	"time"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"
)

func main() {
	sdkKey := os.Getenv("DVC_SERVER_KEY")
	if sdkKey == "" {
		log.Fatal("DVC_SERVER_KEY env var not set: set it to your SDK key")
	}
	variable := os.Getenv("DVC_VARIABLE")
	if variable == "" {
		log.Fatal("DVC_VARIABLE env var not set: set it to a variable key")
	}

	user := devcycle.User{UserId: "test"}
	dvcOptions := devcycle.Options{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         0,
		ConfigPollingIntervalMS:      10 * time.Second,
		RequestTimeout:               10 * time.Second,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
	}

	client, _ := devcycle.NewClient(sdkKey, &dvcOptions)

	log.Printf("client initialized")

	features, _ := client.AllFeatures(user)
	for key, feature := range features {
		log.Printf("Key:%s, feature:%#v", key, feature)
	}

	variables, _ := client.AllVariables(user)
	for key, variable := range variables {
		log.Printf("Key:%s, variable:%#v", key, variable)
	}

	existingVariable, err := client.Variable(user, variable, "DEFAULT")
	if err != nil {
		log.Fatalf("Error getting variable %v: %v", variable, err)
	}
	log.Printf("variable %v: value=%v (%v) defaulted=%t", existingVariable.Key, existingVariable.Value, existingVariable.Type_, existingVariable.IsDefaulted)
	if existingVariable.IsDefaulted {
		log.Printf("Warning: variable %v should not be defaulted", existingVariable.Key)
	}

	variableValue, err := client.VariableValue(user, variable, "DEFAULT")
	if err != nil {
		log.Fatalf("Error getting variable value %v: %v", variableValue, err)
	}
	log.Printf("variable value=%v", variableValue)

	missingVariable, _ := client.Variable(user, variable+"-does-not-exist", "DEFAULT")
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
}
