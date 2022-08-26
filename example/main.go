package main

import (
	"context"
	"fmt"
	"github.com/devcyclehq/go-server-sdk"
	"log"
)

func main() {

	user := devcycle.UserData{UserId: "test"}
	auth := context.WithValue(context.Background(), devcycle.ContextAPIKey, devcycle.APIKey{
		Key: "server-key-here",
	})
	dvcOptions := devcycle.DVCOptions{EnableEdgeDB: false}

	client := devcycle.NewDVCClient("server-key-here", &dvcOptions)
	go func() {
		for {
			select {
			case e := <-client.SDKEventChannel:
				fmt.Println(e)
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

	event := devcycle.Event{
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
