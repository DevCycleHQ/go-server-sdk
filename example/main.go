package main

import (
    "github.com/devcyclehq/go-server-sdk"
    "context"
    "log"
)

func main() {

    user := devcycle.UserData{UserId: "test"}
    auth := context.WithValue(context.Background(), devcycle.ContextAPIKey, devcycle.APIKey{
        Key: "server-key-here",
    })

    client := devcycle.NewDVCClient()
    features, _ := client.DevcycleApi.AllFeatures(auth, user)
	for key, feature := range features {
        log.Printf("Key:%s, feature:%s", key, feature)
    }

    variables, _ := client.DevcycleApi.AllVariables(auth, user)
    for key, variable := range variables {
        log.Printf("Key:%s, feature:%s", key, variable)
    }

    event := devcycle.Event{
      Type_: "customEvent",
      Target: "somevariable.key"}

    vara, _ := client.DevcycleApi.Variable(auth, user, "elliot-test", "test")
    if !vara.IsDefaulted {
        log.Printf("vara not defaulted:%b", vara.IsDefaulted)
    }
    varaDefaulted, _ := client.DevcycleApi.Variable(auth, user, "elliot-asdasd", "test")
    if varaDefaulted.IsDefaulted {
        log.Printf("vara defaulted:%b", varaDefaulted.IsDefaulted)
    }

    response, _ := client.DevcycleApi.Track(auth, user, event)
    log.Printf(response.Message)
}
