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

func setupUserPool() []devcycle.User {
	users := make([]devcycle.User, 10)
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

		users[i] = devcycle.User{
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

var booleanVariables = []string{
	"v-key-1",
	"v-key-2",
	"v-key-5",
	"v-key-6",
	"v-key-7",
	"v-key-11",
	"v-key-12",
	"v-key-13",
	"v-key-15",
	"v-key-17",
	"v-key-22",
	"v-key-23",
	"v-key-24",
	"v-key-25",
	"v-key-26",
	"v-key-28",
	"v-key-29",
	"v-key-30",
	"v-key-32",
	"v-key-33",
	"v-key-36",
	"v-key-37",
	"v-key-38",
	"v-key-39",
	"v-key-40",
	"v-key-41",
	"v-key-42",
	"v-key-44",
	"v-key-45",
	"v-key-46",
	"v-key-47",
	"v-key-50",
	"v-key-51",
	"v-key-52",
	"v-key-53",
	"v-key-54",
	"v-key-58",
	"v-key-59",
	"v-key-61",
	"v-key-62",
	"v-key-63",
	"v-key-64",
	"v-key-65",
	"v-key-66",
	"v-key-68",
	"v-key-70",
	"v-key-71",
	"v-key-72",
	"v-key-73",
	"v-key-74",
	"v-key-75",
	"v-key-77",
	"v-key-78",
	"v-key-79",
	"v-key-80",
	"v-key-81",
	"v-key-85",
}
