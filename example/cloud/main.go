package main

import (
	"context"
	"github.com/devcyclehq/go-server-sdk"
	"log"
	"os"
	"time"
)

func main() {
	environmentKey := os.Getenv("DVC_SERVER_KEY")
	user := devcycle.UserData{UserId: "test"}
	auth := context.WithValue(context.Background(), devcycle.ContextAPIKey, devcycle.APIKey{
		Key: environmentKey,
	})
	dvcOptions := devcycle.DVCOptions{
		EnableEdgeDB:                 false,
		DisableLocalBucketing:        true,
		EventsFlushInterval:          time.Second * 10,
		PollingInterval:              time.Second * 10,
		RequestTimeout:               time.Second * 10,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
	}
	client, _ := devcycle.NewDVCClient(environmentKey, &dvcOptions, nil)

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

	response, _ := client.DevCycleApi.Track(auth, user, event)
	log.Printf(response.Message)
}
