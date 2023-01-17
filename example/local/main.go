package main

import (
	"context"
	"log"
	"os"
	"time"

	devcycle "github.com/devcyclehq/go-server-sdk"
)

func main() {
	environmentKey := os.Getenv("DVC_SERVER_KEY")
	user := devcycle.UserData{UserId: "test"}
	auth := context.WithValue(context.Background(), devcycle.ContextAPIKey, devcycle.APIKey{
		Key: environmentKey,
	})

	dvcOptions := devcycle.DVCOptions{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         0,
		ConfigPollingIntervalMS:      10 * time.Second,
		RequestTimeout:               10 * time.Second,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
	}

	lb, err := devcycle.InitializeLocalBucketing(environmentKey, &dvcOptions)
	if err != nil {
		log.Fatal(err)
	}
	client, _ := devcycle.NewDVCClient(environmentKey, &dvcOptions, lb)

	features, _ := client.DevCycleApi.AllFeatures(auth, user)
	for key, feature := range features {
		log.Printf("Key:%s, feature:%s", key, feature)
	}

	variables, _ := client.DevCycleApi.AllVariables(auth, user)
	for key, variable := range variables {
		log.Printf("Key:%s, feature:%v", key, variable)
	}

	event := devcycle.DVCEvent{
		Type_:  "customEvent",
		Target: "somevariable.key"}

	vara, _ := client.DevCycleApi.Variable(auth, user, "elliot-test", "test")

	if !vara.IsDefaulted {
		log.Printf("vara not defaulted:%v", vara.IsDefaulted)
	}
	varaDefaulted, _ := client.DevCycleApi.Variable(auth, user, "elliot-asdasd", "test")
	if varaDefaulted.IsDefaulted {
		log.Printf("vara defaulted:%v", varaDefaulted.IsDefaulted)
	}

	_, _ = client.DevCycleApi.Track(auth, user, event)
}
