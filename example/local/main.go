package main

import (
	"log"
	"os"
	"time"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"
)

func main() {
	sdkKey := os.Getenv("DVC_SERVER_KEY")
	user := devcycle.DVCUser{UserId: "test"}
	onInitialized := make(chan bool)
	dvcOptions := devcycle.DVCOptions{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         0,
		ConfigPollingIntervalMS:      10 * time.Second,
		RequestTimeout:               10 * time.Second,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
		OnInitializedChannel:         onInitialized,
	}

	client, _ := devcycle.NewDVCClient(sdkKey, &dvcOptions)

	features, _ := client.AllFeatures(user)
	for key, feature := range features {
		log.Printf("Key:%s, feature:%s", key, feature)
	}

	<-onInitialized
	log.Printf("client initialized")

	variables, _ := client.AllVariables(user)
	for key, variable := range variables {
		log.Printf("Key:%s, feature:%v", key, variable)
	}

	event := devcycle.DVCEvent{
		Type_:  "customEvent",
		Target: "somevariable.key"}

	vara, _ := client.Variable(user, "elliot-test", "test")

	if !vara.IsDefaulted {
		log.Printf("vara not defaulted:%v", vara.IsDefaulted)
	}
	varaDefaulted, _ := client.Variable(user, "elliot-asdasd", "test")
	if varaDefaulted.IsDefaulted {
		log.Printf("vara defaulted:%v", varaDefaulted.IsDefaulted)
	}

	_, _ = client.Track(user, event)
}
