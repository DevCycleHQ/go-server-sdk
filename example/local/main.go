package main

import (
	"context"
	"github.com/devcyclehq/go-server-sdk"
	"log"
	"time"
)

func main() {

	user := devcycle.UserData{UserId: "test"}
	auth := context.WithValue(context.Background(), devcycle.ContextAPIKey, devcycle.APIKey{
		Key: "server-key-here",
	})
	dvcOptions := devcycle.DVCOptions{
		EnableEdgeDB:          false,
		DisableLocalBucketing: false,
		PollingInterval:       10 * time.Second,
		RequestTimeout:        10 * time.Second,
		SDKEventsReceiver:     make(chan devcycle.SDKEvent, 100),
	}

	client := devcycle.NewDVCClient("server-key-here", &dvcOptions)
	go func() {
		for {
			select {
			case e := <-client.SDKEventChannel:
				log.Println(e)
			}
		}
	}()
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
