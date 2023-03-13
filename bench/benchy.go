package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"
)

func main() {
	variableKey := os.Getenv("DVC_VARIABLE_KEY")
	userId := os.Getenv("DVC_USER_ID")
	client, err := devcycle.NewDVCClient(os.Getenv("DVC_SERVER_SDK_KEY"), &devcycle.DVCOptions{})
	if err != nil {
		log.Fatalf("Error setting up DVC client: %v", err)
	}
	dvcUser := devcycle.DVCUser{
		UserId: userId,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/variable", func(res http.ResponseWriter, req *http.Request) {
		variable, err := client.Variable(dvcUser, variableKey, false)
		if err != nil {
			log.Printf("Error calling Variable: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(res, "%v", variable.IsDefaulted)
	})
	mux.HandleFunc("/empty", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(200)
	})
	log.Printf("Setting up http server")
	log.Print(http.ListenAndServe(":8080", mux))
}
