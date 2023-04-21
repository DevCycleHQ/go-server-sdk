package native_bucketing

import (
	_ "embed"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/devcyclehq/go-server-sdk/v2/api"
)

var (
	//go:embed testdata/fixture_test_config.json
	test_config []byte
)

func TestUserHashingBucketing_BucketingDistribution(t *testing.T) {
	buckets := map[string]float64{
		"var1":  0,
		"var2":  0,
		"var3":  0,
		"var4":  0,
		"total": 0,
	}

	testTarget := Target{
		Id: "target",
		Audience: &Audience{
			NoIdAudience: NoIdAudience{
				Filters: &AudienceOperator{
					Operator: "and",
				},
			},
			Id: "id",
		},
		Distribution: []TargetDistribution{
			{
				Variation:  "var1",
				Percentage: 0.25,
			},
			{
				Variation:  "var2",
				Percentage: 0.45,
			},
			{
				Variation:  "var3",
				Percentage: 0.1,
			},
			{
				Variation:  "var4",
				Percentage: 0.2,
			},
		},
	}

	for i := 0; i < 30000; i++ {
		userid := uuid.New()
		hash := generateBoundedHashes(userid.String(), testTarget.Id)
		variation, err := testTarget.DecideTargetVariation(hash.BucketingHash)
		if err != nil {
			return
		}
		buckets[variation]++
		buckets["total"]++
	}

	fmt.Println(buckets)
	if !(float64(buckets["var1"]/buckets["total"]) > 0.24) {
		t.Errorf("var1 distribution is not correct: %f", buckets["var1"]/buckets["total"])
	}
	if !(float64(buckets["var1"]/buckets["total"]) < 0.26) {
		t.Errorf("var1 distribution is not correct %f", buckets["var1"]/buckets["total"])
	}
	if !(float64(buckets["var2"]/buckets["total"]) > 0.44) {
		t.Errorf("var2 distribution is not correct %f", buckets["var2"]/buckets["total"])
	}
	if !(float64(buckets["var2"]/buckets["total"]) < 0.46) {
		t.Errorf("var2 distribution is not correct %f", buckets["var2"]/buckets["total"])
	}
	if !(float64(buckets["var3"]/buckets["total"]) > 0.09) {
		t.Errorf("var3 distribution is not correct %f", buckets["var3"]/buckets["total"])
	}
	if !(float64(buckets["var3"]/buckets["total"]) < 0.11) {
		t.Errorf("var3 distribution is not correct %f", buckets["var3"]/buckets["total"])
	}
	if !(float64(buckets["var4"]/buckets["total"]) > 0.19) {
		t.Errorf("var4 distribution is not correct %f", buckets["var4"]/buckets["total"])
	}
	if !(float64(buckets["var4"]/buckets["total"]) < 0.21) {
		t.Errorf("var4 distribution is not correct %f", buckets["var4"]/buckets["total"])
	}
}

func TestBucketing_Deterministic_SameUserSameSeed(t *testing.T) {
	userId := uuid.New()
	hash := generateBoundedHashes(userId.String(), "fake")
	hash2 := generateBoundedHashes(userId.String(), "fake")
	if hash.BucketingHash != hash2.BucketingHash {
		t.Errorf("Hashes should be the same for the same target id and userid")
	}

	if hash.RolloutHash != hash2.RolloutHash {
		t.Errorf("Hashes should be the same for the same target id and userid")
	}
}

func TestBucketing_Deterministic_SameUserDiffSeed(t *testing.T) {
	userId := uuid.New()
	hash := generateBoundedHashes(userId.String(), "fake")
	hash2 := generateBoundedHashes(userId.String(), "fake2")
	if hash.BucketingHash == hash2.BucketingHash {
		t.Errorf("Hashes should be different for different target ids")
	}
}

func TestBucketing_Deterministic_RolloutNotEqualBucketing(t *testing.T) {
	userId := uuid.New()
	hash := generateBoundedHashes(userId.String(), "fake")
	if hash.BucketingHash == hash.RolloutHash {
		t.Errorf("Hashes should be different - rollout should not equal bucketing hash")
	}
}

func TestConfigParsing(t *testing.T) {
	// Parsing the large config should succeed without an error
	err := SetConfig(test_config, "test", "")
	require.NoError(t, err)
	config, err := getConfig("test")
	require.NoError(t, err)

	// Spot check parsing down to a filter
	features := config.Features
	require.Len(t, features, 4)
	targets := features[0].Configuration.Targets
	require.Len(t, targets, 3)
	filters := targets[0].Audience.Filters.Filters
	require.Len(t, filters, 1)
	require.Equal(t, "user", filters[0].GetType())
	require.Equal(t, "email", filters[0].GetSubType())
	require.Equal(t, "=", filters[0].GetComparator())

	// Check maps of variables IDs and keys
	require.Equal(t, map[string]Variable{
		"614ef6ea475129459160721a": {Id: "614ef6ea475129459160721a", Type: "String", Key: "test"},
		"615356f120ed334a6054564c": {Id: "615356f120ed334a6054564c", Type: "String", Key: "swagTest"},
		"61538237b0a70b58ae6af71f": {Id: "61538237b0a70b58ae6af71f", Type: "String", Key: "feature2Var"},
		"61538237b0a70b58ae6af71g": {Id: "61538237b0a70b58ae6af71g", Type: "String", Key: "feature2.cool"},
		"61538237b0a70b58ae6af71h": {Id: "61538237b0a70b58ae6af71h", Type: "String", Key: "feature2.hello"},
		"61538237b0a70b58ae6af71q": {Id: "61538237b0a70b58ae6af71q", Type: "JSON", Key: "json-var"},
		"61538237b0a70b58ae6af71s": {Id: "61538237b0a70b58ae6af71s", Type: "Number", Key: "num-var"},
		"61538237b0a70b58ae6af71y": {Id: "61538237b0a70b58ae6af71y", Type: "Boolean", Key: "bool-var"},
		"61538237b0a70b58ae6af71z": {Id: "61538237b0a70b58ae6af71z", Type: "String", Key: "audience-match"},
		"61538937b0a70b58ae6af71f": {Id: "61538937b0a70b58ae6af71f", Type: "String", Key: "feature4Var"}},
		config.variableIdMap,
	)
	require.Equal(t, map[string]Variable{
		"audience-match": {Id: "61538237b0a70b58ae6af71z", Type: "String", Key: "audience-match"},
		"bool-var":       {Id: "61538237b0a70b58ae6af71y", Type: "Boolean", Key: "bool-var"},
		"feature2.cool":  {Id: "61538237b0a70b58ae6af71g", Type: "String", Key: "feature2.cool"},
		"feature2.hello": {Id: "61538237b0a70b58ae6af71h", Type: "String", Key: "feature2.hello"},
		"feature2Var":    {Id: "61538237b0a70b58ae6af71f", Type: "String", Key: "feature2Var"},
		"feature4Var":    {Id: "61538937b0a70b58ae6af71f", Type: "String", Key: "feature4Var"},
		"json-var":       {Id: "61538237b0a70b58ae6af71q", Type: "JSON", Key: "json-var"},
		"num-var":        {Id: "61538237b0a70b58ae6af71s", Type: "Number", Key: "num-var"},
		"swagTest":       {Id: "615356f120ed334a6054564c", Type: "String", Key: "swagTest"},
		"test":           {Id: "614ef6ea475129459160721a", Type: "String", Key: "test"}},
		config.variableKeyMap,
	)
}

func TestRollout_Gradual(t *testing.T) {
	rollout := Rollout{
		Type:            "gradual",
		StartPercentage: 0,
		StartDate:       time.Now().Add(time.Hour * -24),
		Stages: []RolloutStage{
			{
				Type:       "linear",
				Date:       time.Now().Add(time.Hour * 24),
				Percentage: 1,
			},
		},
	}
	if !doesUserPassRollout(rollout, 0.35) {
		t.Errorf("User should pass rollout - 0.35")
	}
	if doesUserPassRollout(rollout, 0.85) {
		t.Errorf("User should not pass rollout - 0.85")
	}
	if !doesUserPassRollout(rollout, 0.2) {
		t.Errorf("User should pass rollout - 0.2")
	}
	if doesUserPassRollout(rollout, 0.75) {
		t.Errorf("User should not pass rollout - 0.75")
	}
	t.Log("Changing rollout percentage to 0.8")
	rollout.Stages[0].Percentage = 0.8

	if doesUserPassRollout(rollout, 0.51) {
		t.Error("User should not pass rollout - 0.51")
	}

	if doesUserPassRollout(rollout, 0.95) {
		t.Error("User should not pass rollout - 0.95")
	}

	if !doesUserPassRollout(rollout, 0.35) {
		t.Error("User should pass rollout - 0.35")
	}
}

func TestRollout_Gradual_WithStartDateFuture(t *testing.T) {
	rollout := Rollout{
		Type:            "gradual",
		StartPercentage: 0,
		StartDate:       time.Now().Add(time.Hour * 24),
		Stages: []RolloutStage{
			{
				Type:       "linear",
				Date:       time.Now().Add(time.Hour * 48),
				Percentage: 1,
			},
		},
	}

	if doesUserPassRollout(rollout, 0) {
		t.Error("User should not pass rollout - 0")
	}
	if doesUserPassRollout(rollout, 0.25) {
		t.Error("User should not pass rollout - 0.25")
	}
	if doesUserPassRollout(rollout, 0.5) {
		t.Error("User should not pass rollout - 0.5")
	}
	if doesUserPassRollout(rollout, 0.75) {
		t.Error("User should not pass rollout - 0.75")
	}
	if doesUserPassRollout(rollout, 1) {
		t.Error("User should not pass rollout - 1")
	}
}

func TestRollout_Gradual_WithStartDate_NoEnd(t *testing.T) {
	rollout := Rollout{
		Type:            "gradual",
		StartPercentage: 1,
		StartDate:       time.Now().Add(time.Hour * -24),
		Stages:          []RolloutStage{},
	}

	if !doesUserPassRollout(rollout, 0) {
		t.Error("User should pass rollout - 0")
	}
	if !doesUserPassRollout(rollout, 0.25) {
		t.Error("User should pass rollout - 0.25")
	}
	if !doesUserPassRollout(rollout, 0.5) {
		t.Error("User should pass rollout - 0.5")
	}
	if !doesUserPassRollout(rollout, 0.75) {
		t.Error("User should pass rollout - 0.75")
	}
	if !doesUserPassRollout(rollout, 1) {
		t.Error("User should pass rollout - 1")
	}
}

func TestRollout_Gradual_WithStartDate_NoEnd_Future(t *testing.T) {
	rollout := Rollout{
		Type:            "gradual",
		StartPercentage: 0,
		StartDate:       time.Now().Add(time.Hour * 24),
		Stages:          []RolloutStage{},
	}

	if doesUserPassRollout(rollout, 0) {
		t.Error("User should not pass rollout - 0")
	}
	if doesUserPassRollout(rollout, 0.25) {
		t.Error("User should not pass rollout - 0.25")
	}
	if doesUserPassRollout(rollout, 0.5) {
		t.Error("User should not pass rollout - 0.5")
	}
	if doesUserPassRollout(rollout, 0.75) {
		t.Error("User should not pass rollout - 0.75")
	}
	if doesUserPassRollout(rollout, 1) {
		t.Error("User should not pass rollout - 1")
	}
}

func TestRollout_Schedule_Valid(t *testing.T) {
	rollout := Rollout{
		Type:      "schedule",
		StartDate: time.Now().Add(time.Minute * -1),
	}

	if !doesUserPassRollout(rollout, 0) {
		t.Error("User should pass rollout - 0")
	}
	if !doesUserPassRollout(rollout, 0.25) {
		t.Error("User should pass rollout - 0.25")
	}
	if !doesUserPassRollout(rollout, 0.5) {
		t.Error("User should pass rollout - 0.5")
	}
	if !doesUserPassRollout(rollout, 0.75) {
		t.Error("User should pass rollout - 0.75")
	}
	if !doesUserPassRollout(rollout, 1) {
		t.Error("User should pass rollout - 1")
	}
}

func TestRollout_Schedule_Future(t *testing.T) {
	rollout := Rollout{
		Type: "schedule",

		StartDate: time.Now().Add(time.Minute * 1),
	}

	if doesUserPassRollout(rollout, 0) {
		t.Error("User should not pass rollout - 0")
	}
	if doesUserPassRollout(rollout, 0.25) {
		t.Error("User should not pass rollout - 0.25")
	}
	if doesUserPassRollout(rollout, 0.5) {
		t.Error("User should not pass rollout - 0.5")
	}
	if doesUserPassRollout(rollout, 0.75) {
		t.Error("User should not pass rollout - 0.75")
	}
	if doesUserPassRollout(rollout, 1) {
		t.Error("User should not pass rollout - 1")
	}
}

func TestRollout_Stepped_Valid(t *testing.T) {
	rollout := Rollout{
		Type: "stepped",
		Stages: []RolloutStage{
			{
				Type:       "discrete",
				Date:       time.Now().Add(time.Hour * -48),
				Percentage: 0.25,
			},
			{
				Type:       "discrete",
				Date:       time.Now().Add(time.Hour * -24),
				Percentage: 0.5,
			},
			{
				Type:       "discrete",
				Date:       time.Now().Add(time.Hour * 24),
				Percentage: 0.75,
			},
		},
	}

	if !doesUserPassRollout(rollout, 0) {
		t.Error("User should pass rollout - 0")
	}
	if !doesUserPassRollout(rollout, 0.25) {
		t.Error("User should pass rollout - 0.25")
	}
	if !doesUserPassRollout(rollout, 0.4) {
		t.Error("User should pass rollout - 0.4")
	}
	if doesUserPassRollout(rollout, 0.6) {
		t.Error("User should not pass rollout - 0.6")
	}
	if doesUserPassRollout(rollout, 0.9) {
		t.Error("User should not pass rollout - 0.9")
	}
}

func TestRollout_Stepped_Error(t *testing.T) {
	rollout := Rollout{}
	if doesUserPassRollout(rollout, 0) {
		t.Error("User should not pass rollout - empty")
	}
	if doesUserPassRollout(rollout, 1) {
		t.Error("User should not pass rollout - empty")
	}
}

func TestClientData(t *testing.T) {
	user := api.DVCUser{
		UserId: "client-test",
		CustomData: map[string]interface{}{
			"favouriteFood": "pizza",
			"favouriteNull": nil,
		},
	}.GetPopulatedUser(&api.PlatformData{
		PlatformVersion: "1.1.2",
	})

	err := SetConfig(test_config, "test", "")
	require.NoError(t, err)

	// Ensure bucketed config has a feature variation map that's empty
	bucketedUserConfig, err := GenerateBucketedConfig("test", user, nil)
	require.NoError(t, err)
	variableUser, err := generateBucketedVariableForUser("test", user, "num-var", nil)
	require.ErrorContainsf(t, err, "does not qualify", "does not qualify")
	require.Nil(t, variableUser)
	require.Equal(t, map[string]string{}, bucketedUserConfig.FeatureVariationMap)

	clientCustomData := map[string]interface{}{
		"favouriteFood":  "NOT PIZZA!!",
		"favouriteDrink": "coffee",
	}

	bucketedUserConfig, err = GenerateBucketedConfig("test", user, clientCustomData)
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"614ef6aa473928459060721a": "615357cf7e9ebdca58446ed0",
		"614ef6aa475928459060721a": "615382338424cb11646d7667",
	}, bucketedUserConfig.FeatureVariationMap)
	variableUser, err = generateBucketedVariableForUser("test", user, "num-var", clientCustomData)
	require.NoError(t, err)
	require.Equal(t, 610.61, variableUser.Variable.Value)

	user2 := api.DVCUser{
		UserId: "hates-pizza",
		CustomData: map[string]interface{}{
			"favouriteFood": "NOT PIZZA!",
		},
	}.GetPopulatedUser(&api.PlatformData{
		PlatformVersion: "1.1.2",
	})
	bucketedUserConfig, err = GenerateBucketedConfig("test", user2, nil)
	require.NoError(t, err)

	require.Equal(t, map[string]string{}, bucketedUserConfig.FeatureVariationMap)

}

func TestVariableForUser(t *testing.T) {
	user := api.DVCUser{
		UserId: "CPopultest",
		CustomData: map[string]interface{}{
			"favouriteDrink": "coffee",
			"favouriteFood":  "pizza",
		},
	}.GetPopulatedUser(&api.PlatformData{
		PlatformVersion: "1.1.2",
	})

	err := SetConfig(test_config, "test", "")
	require.NoError(t, err)

	userVariable, err := generateBucketedVariableForUser("test", user, "json-var", nil)
	require.NoError(t, err)
	require.Equal(t, "615357cf7e9ebdca58446ed0", userVariable.Variation.Id)
	require.Equal(t, "{\"hello\":\"world\",\"num\":610,\"bool\":true}", userVariable.Variable.Value)

}
