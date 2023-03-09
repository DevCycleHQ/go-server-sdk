package main

import (
	devcycle "github.com/devcyclehq/go-server-sdk/v2"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

func main() {

	variableKey := os.Getenv("DVC_VARIABLE_KEY")
	userId := os.Getenv("DVC_USER_ID")
	client, err := devcycle.NewDVCClient(os.Getenv("DVC_SERVER_SDK_KEY"), &devcycle.DVCOptions{})
	if err != nil {
		return
	}
	dvcUser := devcycle.DVCUser{
		UserId: userId,
	}
	r := gin.Default()
	r.GET("/variable", func(c *gin.Context) {
		variable, err := client.Variable(dvcUser, variableKey, false)
		if err != nil {
			return
		}
		c.String(200, strconv.FormatBool(variable.IsDefaulted))
	})
	r.GET("/empty", func(c *gin.Context) {
		c.String(200, "")
	})
	r.Run()
}
