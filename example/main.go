package main

import (
    "github.com/devcyclehq/go-server-sdk"
    "context"
    "log"
)

func main() {

    user := devcycle.UserData{UserId: "test"}
    auth := context.WithValue(context.Background(), devcycle.ContextAPIKey, devcycle.APIKey{
        Key: "server-1bf0c139-6861-41e1-8d2d-ea0416045f99",
    })

    conf := devcycle.NewConfiguration()
    client := devcycle.NewDVCClient(conf)
    features, _ := client.DevcycleApi.GetFeatures(auth, user)
	for key, feature := range features {
        log.Printf("Key:%s, feature:%s", key, feature)
    }

    variables, _ := client.DevcycleApi.GetVariables(auth, user)
    for key, variable := range variables {
        log.Printf("Key:%s, feature:%s", key, variable)
    }

    event := devcycle.Event{
      Type_: "customEvent",
      Target: "somevariable.key"}

    vara, _ := client.DevcycleApi.GetVariableByKey(auth, user, "elliot-asdasd", "test")
    log.Printf("vara:%s", vara)

    response, _ := client.DevcycleApi.PostEvent(auth, user, event)
    log.Printf(response.Message)
}
