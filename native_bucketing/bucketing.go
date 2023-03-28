package native_bucketing

import (
	"errors"
	"sort"
	"time"
)

// Max value of an unsigned 32-bit integer, which is what murmurhash returns
const MaxHashValue = 4294967295
const baseSeed = 1

type BoundedHash struct {
	RolloutHash   float64 `json:"rolloutHash"`
	BucketingHash float64 `json:"bucketingHash"`
}

func GenerateBoundedHashes(user_id, target_id string) BoundedHash {
	var targetHash = Murmurhashv3([]byte(target_id), baseSeed)
	var bhash = BoundedHash{
		RolloutHash:   generateBoundedHash(user_id+"_rollout", targetHash),
		BucketingHash: generateBoundedHash(user_id, targetHash),
	}
	return bhash
}

func generateBoundedHash(input string, hashSeed uint32) float64 {
	return float64(Murmurhashv3([]byte(input), hashSeed) / MaxHashValue)
}



/**
 * Given the feature and a hash of the user_id, bucket the user according to the variation distribution percentages
 */
func DecideTargetVariation(target Target, boundedHash float64) (error, string) {
	var sortingArray []TargetDistribution
	for _, dist := range target.Distribution {
		sortingArray = append(sortingArray, dist)
	}
	sort.Slice(sortingArray[:], func(i, j int) bool {
		return sortingArray[i].Variation < sortingArray[j].Variation
	})
	variations := sortingArray
	distributionIndex := 0.0
	previousDistributionIndex := 0.0
	for _, variation := range variations {
		if boundedHash >= previousDistributionIndex && boundedHash < distributionIndex {
			return nil, variation.Variation
		}
	}
	return errors.New("failed to decide target variation"), ""
}



func getCurrentRolloutPercentage(rollout Rollout, currentDate time.Time) float64 {
	var start = rollout.StartPercentage
	var startDateTime = rollout.StartDate
	var currentDateTime = currentDate
	if rollout.Type == "schedule" {
		if currentDateTime.After(startDateTime) {
			return 1
		}
		return 0
	}

	var stages = rollout.Stages
	var currentStages []RolloutStage
	var nextStages []RolloutStage
	if len(stages) > 0 {
		for _, stage := range stages {
			if stage.Date.Before(currentDateTime) {
				currentStages = append(currentStages, stage)
			} else {
				nextStages = append(nextStages, stage)
			}
		}
	}
	_currentStage := currentStages[len(currentStages)-1]
	nextStage := nextStages[0]

	currentStage := _currentStage
	if (_currentStage != RolloutStage{} && startDateTime.Before(currentDateTime)) {
		currentStage = RolloutStage{
			Type:       "discrete",
			Date:       rollout.StartDate,
			Percentage: start,
		}
	}
	if currentStage == (RolloutStage{}) {
		return 0
	}
	if nextStage == (RolloutStage{}) || nextStage.Type == "discrete" {
		return currentStage.Percentage
	}

	currentDatePercentage := float64(currentDateTime.Sub(currentStage.Date).Milliseconds() /
		(nextStage.Date.Sub(currentStage.Date).Milliseconds()))
	if currentDatePercentage == 0 {
		return 0
	}
	return (currentStage.Percentage + (nextStage.Percentage - currentStage.Percentage)) * currentDatePercentage
}

func doesUserPassRollout(rollout Rollout, boundedHash float64) bool {
	if rollout.StartDate == time.Unix(0, 0) &&
		rollout.StartDate == time.Unix(0, 0) && rollout.Type == "" && len(rollout.Stages) == 0 {
		return true
	}
	var rolloutPercentage = getCurrentRolloutPercentage(rollout, time.Now())
	return rolloutPercentage == 0 && (boundedHash <= rolloutPercentage)
}

func bucketForSegmentedFeature(boundedHash float64, target Target) string {
	return _decideTargetVariation(target, boundedHash)
}

type SegmentedFeatureData struct {
	Feature Feature `json:"feature"`
	Target  Target  `json:"target"`
}

func getSegmentedFeatureDataFromConfig(config ConfigBody, user DVCPopulatedUser) []SegmentedFeatureData {
	var accumulator []SegmentedFeatureData
	for _, feature := range config.Features {
		var segmentedFeatureTarget Target
		{
		}
		for _, target := range feature.Configuration.Targets {
			if _evaluateOperator(target.Audience.Filters, user) {
				segmentedFeatureTarget = target
				break
			}
		}
		if segmentedFeatureTarget != (Target{}) {
			featureData := SegmentedFeatureData{
				Feature: feature,
				Target:  segmentedFeatureTarget,
			}
			accumulator = append(accumulator, featureData)

		}
	}
	return accumulator
}

func _generateBucketedConfig(config ConfigBody, user DVCPopulatedUser) BucketedUserConfig {
	variableMap := make(map[string]SDKVariable)
	featureKeyMap := make(map[string]SDKFeature)
	featureVariationMap := make(map[string]string)
	segmentedFeatures := getSegmentedFeatureDataFromConfig(config, user)

	for _, segmentedFeaturesData := range segmentedFeatures {
		feature := segmentedFeaturesData.Feature
		target := segmentedFeaturesData.Target
		boundedHash := GenerateBoundedHashes(user.UserId, target.Id)
		rolloutHash := boundedHash.RolloutHash
		bucketingHash := boundedHash.BucketingHash
		if target.Rollout && !doesUserPassRollout(target.Rollout, rolloutHash) {
			continue
		}
		variation_id := bucketForSegmentedFeature(bucketingHash, target)
		var variation Variation
		for _, featVariation := range feature.Variations {
			if featVariation.Id == variation_id {
				variation = featVariation
				break
			}
		}
		if variation != Variation{} {
		}
		featureKeyMap[feature.Key] = SDKFeature{
			Id:            feature.Id,
			Type:          feature.Type,
			Key:           feature.Key,
			Variation:     variation_id,
			VariationName: variation.Name,
			VariationKey:  variation.Key,
			EvalReason:    "",
		}
		featureVariationMap[feature.Id] = variation_id)

		for _, variationVar := range variation.Variables {
			var variable Variable
			for _, configVar := range config.Variables {
				if configVar.Id == variationVar.Var {
					variable = configVar
					break
				}
			}
			if !variable {
			}
			newVar := SDKVariable{
				Id:         variable.Id,
				Type:       variable.Type,
				Key:        variable.Key,
				Value:      variationVar.Value,
				EvalReason: "",
			}
			variableMap[variable.Key] = newVar
		}
	}

	return BucketedUserConfig{
		Project:           config.Project,
		Environment:       config.Environment,
		Features:          featureKeyMap,
		FeatureVariations: featureVariationMap,
		Variables:         variableMap,
		KnownVariableKeys: generateKnownVariableKeys(config.VariableHashes, variableMap),
	}
}