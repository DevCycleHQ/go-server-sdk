package main

import (
	_ "embed"
	"fmt"
	"math/rand"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"
)

var (
	//go:embed testdata/fixture_large_config.json
	test_large_config          []byte
	test_large_config_variable = "v-key-25"
	lettersAndNumbers          = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = lettersAndNumbers[rand.Intn(len(lettersAndNumbers))]
	}
	return string(b)
}

func setupUserPool() []devcycle.DVCUser {
	users := make([]devcycle.DVCUser, 10)
	for i := 0; i < 10; i++ {
		customData := map[string]interface{}{
			"cacheKey":   randSeq(250),
			"propInt":    rand.Intn(1000),
			"propDouble": rand.Float32(),
			"propBool":   rand.Intn(2) == 1,
		}

		customPrivateData := map[string]interface{}{
			"aPrivateValue": "secret-data-here",
		}

		users[i] = devcycle.DVCUser{
			UserId:            fmt.Sprintf("user_%d", i),
			DeviceModel:       "testing",
			Name:              fmt.Sprintf("Testing User %d", i),
			Email:             fmt.Sprintf("test.user-%d@gmail.com", i),
			AppBuild:          "1.0.0",
			AppVersion:        "0.0.1",
			Country:           "ca",
			Language:          "en",
			CustomData:        customData,
			PrivateCustomData: customPrivateData,
		}
	}
	return users
}
