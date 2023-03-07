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
		EnableCloudBucketing:         true,
		EventFlushIntervalMS:         time.Second * 10,
		ConfigPollingIntervalMS:      time.Second * 10,
		RequestTimeout:               time.Second * 10,
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

	vara, _ := client.Variable(context.Background(), user, "elliot-test", "test")
	if !vara.IsDefaulted {
		log.Printf("vara not defaulted:%v", vara.IsDefaulted)
	}
	varaDefaulted, _ := client.Variable(context.Background(), user, "elliot-asdasd", "test")
	if varaDefaulted.IsDefaulted {
		log.Printf("vara defaulted:%v", varaDefaulted.IsDefaulted)
	}

	_, _ = client.Track(context.Background(), user, event)
}
