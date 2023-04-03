package native_bucketing

import (
	"fmt"
	"time"
)

// Max value of an unsigned 32-bit integer, which is what murmurhash returns
const MaxHashValue uint32 = 4294967295
const baseSeed = 1

type BoundedHash struct {
	RolloutHash   float64 `json:"rolloutHash"`
	BucketingHash float64 `json:"bucketingHash"`
}

func GenerateBoundedHashes(userId, targetId string) BoundedHash {
	var targetHash = murmurhashV3([]byte(targetId), baseSeed)
	var bhash = BoundedHash{
		RolloutHash:   generateBoundedHash(userId+"_rollout", targetHash),
		BucketingHash: generateBoundedHash(userId, targetHash),
	}
	return bhash
}

func generateBoundedHash(input string, hashSeed uint32) float64 {
	mh := murmurhashV3([]byte(input), hashSeed)
	return float64(mh) / float64(MaxHashValue)
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

	for _, stage := range stages {
		if stage.Date.Before(currentDateTime) {
			currentStages = append(currentStages, stage)
		} else {
			nextStages = append(nextStages, stage)
		}
	}

	var _currentStage *RolloutStage
	var nextStage *RolloutStage
	if len(currentStages) == 0 {
		_currentStage = nil
	} else {
		_currentStage = &currentStages[len(currentStages)-1]
	}
	if len(nextStages) == 0 {
		nextStage = nil
	} else {
		nextStage = &nextStages[0]
	}
	currentStage := _currentStage
	if _currentStage == nil && startDateTime.Before(currentDateTime) {
		currentStage = &RolloutStage{
			Type:       "discrete",
			Date:       rollout.StartDate,
			Percentage: start,
		}
	}
	if currentStage == nil {
		return 0
	}
	if nextStage == nil || nextStage.Type == "discrete" {
		return currentStage.Percentage
	}

	currentDatePercentage := float64(currentDateTime.Sub(currentStage.Date).Milliseconds()) /
		float64(nextStage.Date.Sub(currentStage.Date).Milliseconds())
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
	return rolloutPercentage != 0 && (boundedHash <= rolloutPercentage)
}

type SegmentedFeatureData struct {
	Feature Feature `json:"feature"`
	Target  Target  `json:"target"`
}

func evaluateSegmentationForFeature(config ConfigBody, feature Feature, user DVCPopulatedUser, clientCustomData map[string]interface{}) (error, Target) {
	if len(feature.Configuration.Targets) == 0 {
		return fmt.Errorf("feature %s has no targets", feature.Key), Target{}
	}
	for _, target := range feature.Configuration.Targets {
		if _evaluateOperator(target.Audience.Filters, config.Audiences, user, clientCustomData) {
			return nil, target
		}
	}
	return fmt.Errorf("user %s does not qualify for any targets for feature %s", user.UserId, feature.Key), Target{}
}

func getSegmentedFeatureDataFromConfig(config ConfigBody, user DVCPopulatedUser, clientCustomData map[string]interface{}) []SegmentedFeatureData {
	var accumulator []SegmentedFeatureData
	for _, feature := range config.Features {
		var segmentedFeatureTarget *Target

		if err, t2 := evaluateSegmentationForFeature(config, feature, user, clientCustomData); err != nil {
			segmentedFeatureTarget = &t2
			break
		}
		if segmentedFeatureTarget != nil {
			featureData := SegmentedFeatureData{
				Feature: feature,
				Target:  *segmentedFeatureTarget,
			}
			accumulator = append(accumulator, featureData)
		}
	}
	return accumulator
}

type TargetAndHashes struct {
	Target Target
	Hashes BoundedHash
}

func doesUserQualifyForFeature(config ConfigBody, feature Feature, user DVCPopulatedUser, clientCustomData map[string]interface{}) (error, TargetAndHashes) {
	err, target := evaluateSegmentationForFeature(config, feature, user, clientCustomData)
	if err != nil {
		return fmt.Errorf("user %s does not qualify for any targets for feature %s", user.UserId, feature.Key), TargetAndHashes{}
	}

	boundedHashes := GenerateBoundedHashes(user.UserId, target.Id)
	rolloutHash := boundedHashes.RolloutHash

	if target.Rollout != nil && !doesUserPassRollout(*target.Rollout, rolloutHash) {
		return fmt.Errorf("user %s does not qualify for feature %s rollout", user.UserId, feature.Key), TargetAndHashes{}
	}
	return nil, TargetAndHashes{
		Target: target,
		Hashes: boundedHashes,
	}
}

func bucketUserForVariation(feature Feature, hashes TargetAndHashes) (error, Variation) {
	variationId, err := hashes.Target.DecideTargetVariation(hashes.Hashes.BucketingHash)
	if err != nil {
		return err, Variation{}
	}
	var variation *Variation
	for _, v := range feature.Variations {
		if v.Id == variationId {
			variation = &v
		}
	}
	if variation == nil {
		return fmt.Errorf("config missing variation %s", variationId), Variation{}
	}
	return nil, *variation
}

func _generateBucketedConfig(config ConfigBody, user DVCPopulatedUser, clientCustomData map[string]interface{}) BucketedUserConfig {
	variableMap := make(map[string]SDKVariable)
	featureKeyMap := make(map[string]SDKFeature)
	featureVariationMap := make(map[string]string)
	variableVariationMap := make(map[string]FeatureVariation)

	for _, feature := range config.Features {
		err, targetAndHashes := doesUserQualifyForFeature(config, feature, user, clientCustomData)
		if err != nil {
			continue
		}

		err, variation := bucketUserForVariation(feature, targetAndHashes)
		if err != nil {
			continue
		}
		featureKeyMap[feature.Key] = SDKFeature{
			Id:            feature.Id,
			Type:          feature.Type,
			Key:           feature.Key,
			VariationId:   variation.Id,
			VariationKey:  variation.Key,
			VariationName: variation.Name,
		}
		featureVariationMap[feature.Id] = variation.Id

		for _, variationVar := range variation.Variables {
			variable := config.GetVariableForId(variationVar.Var)
			if variable == nil {
				panic(fmt.Sprintf("Config missing variable: %s", variationVar.Var))
			}

			variableVariationMap[variable.Key] = FeatureVariation{
				Variation: variation.Id,
				Feature:   feature.Id,
			}
			newVar := SDKVariable{
				Id:    variable.Id,
				Type:  variable.Type,
				Key:   variable.Key,
				Value: variationVar.Value,
			}
			variableMap[variable.Key] = newVar
		}
	}

	return BucketedUserConfig{
		Project:            config.Project,
		Environment:        config.Environment,
		Features:           featureKeyMap,
		FeatureVariations:  featureVariationMap,
		VariableVariations: variableVariationMap,
		Variables:          variableMap,
	}
}
