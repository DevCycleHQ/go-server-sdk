package main

import (
	"context"
	"log"
	"os"
	"time"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"
)

func main() {
	sdkKey := os.Getenv("DVC_SERVER_KEY")
	user := devcycle.DVCUser{UserId: "test"}
	dvcOptions := devcycle.DVCOptions{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         0,
		ConfigPollingIntervalMS:      10 * time.Second,
		RequestTimeout:               10 * time.Second,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
	}

	client, _ := devcycle.NewDVCClient(sdkKey, &dvcOptions)

	features, _ := client.AllFeatures(context.Background(), user)
	for key, feature := range features {
		log.Printf("Key:%s, feature:%s", key, feature)
	}

	variables, _ := client.AllVariables(context.Background(), user)
	for key, variable := range variables {
		log.Printf("Key:%s, feature:%v", key, variable)
	}

	event := devcycle.DVCEvent{
		Type_:  "customEvent",
		Target: "somevariable.key"}

	for i := 0; i < 100; i++ {
		go evalVariable(client, user)
	}

	_, _ = client.Track(context.Background(), user, event)
	time.Sleep(10 * time.Second)
}

func evalVariable(client *devcycle.DVCClient, user devcycle.DVCUser) (devcycle.Variable, error) {
	vara, err := client.Variable(context.Background(), user, "test", false)
	if vara.Value != true {
		log.Printf("vara not true\n")
	}
	return vara, err
}
