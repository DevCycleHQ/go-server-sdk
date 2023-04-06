package native_bucketing

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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
		hash := GenerateBoundedHashes(userid.String(), testTarget.Id)
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
	hash := GenerateBoundedHashes(userId.String(), "fake")
	hash2 := GenerateBoundedHashes(userId.String(), "fake")
	if hash.BucketingHash != hash2.BucketingHash {
		t.Errorf("Hashes should be the same for the same target id and userid")
	}

	if hash.RolloutHash != hash2.RolloutHash {
		t.Errorf("Hashes should be the same for the same target id and userid")
	}
}

func TestBucketing_Deterministic_SameUserDiffSeed(t *testing.T) {
	userId := uuid.New()
	hash := GenerateBoundedHashes(userId.String(), "fake")
	hash2 := GenerateBoundedHashes(userId.String(), "fake2")
	if hash.BucketingHash == hash2.BucketingHash {
		t.Errorf("Hashes should be different for different target ids")
	}
}

func TestBucketing_Deterministic_RolloutNotEqualBucketing(t *testing.T) {
	userId := uuid.New()
	hash := GenerateBoundedHashes(userId.String(), "fake")
	if hash.BucketingHash == hash.RolloutHash {
		t.Errorf("Hashes should be different - rollout should not equal bucketing hash")
	}
}

func TestConfigParsing(t *testing.T) {
	var config ConfigBody

	// Parsing the large config should succeed without an error
	err := json.Unmarshal([]byte(test_large_config), &config)
	require.NoError(t, err)

	// Spot check parsing down to a filter
	features := config.Features
	require.Len(t, features, 79)
	targets := features[0].Configuration.Targets
	require.Len(t, targets, 2)
	filters := targets[0].Audience.Filters.Filters
	require.Len(t, filters, 1)
	require.Equal(t, "user", filters[0].GetType())
	require.Equal(t, "user_id", filters[0].GetSubType())
	require.Equal(t, "=", filters[0].GetComparator())
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
	t.Skip()
	// Need to serialize a config/generate a bucketed config
	//
	//user := DVCPopulatedUser{
	//	UserId: "client-test",
	//
	//	CustomData: map[string]interface{}{
	//
	//		"favouriteFood": "pizza",
	//		"favouriteNull": nil,
	//	},
	//	PlatformData: PlatformData{
	//		PlatformVersion: "1.1.2",
	//	},
	//}
	//clientCustomData := map[string]interface{}{
	//	"favouriteFood":  "NOT PIZZAA!!",
	//	"favouriteDrink": "coffee",
	//}
	//initSDK
	//_generateBucketedConfig()
	// Ensure bucketed config has a featurefvariationmap that's empty
	// set client custom data
	// generatebucketed config
	// ensure bucketed config has a featurefvariationmap that's not empty and matches
}
