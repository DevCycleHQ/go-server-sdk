package native_bucketing

import (
	"fmt"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
)

// Max value of an unsigned 32-bit integer, which is what murmurhash returns
const maxHashValue uint32 = 4294967295
const baseSeed = 1

type boundedHash struct {
	RolloutHash   float64 `json:"rolloutHash"`
	BucketingHash float64 `json:"bucketingHash"`
}

func generateBoundedHashes(userId, targetId string) boundedHash {
	var targetHash = murmurhashV3([]byte(targetId), baseSeed)
	var bhash = boundedHash{
		RolloutHash:   generateBoundedHash(userId+"_rollout", targetHash),
		BucketingHash: generateBoundedHash(userId, targetHash),
	}
	return bhash
}

func generateBoundedHash(input string, hashSeed uint32) float64 {
	mh := murmurhashV3([]byte(input), hashSeed)
	return float64(mh) / float64(maxHashValue)
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
	if rollout.StartDate == time.Unix(0, 0) && rollout.Type == "" && len(rollout.Stages) == 0 {
		return true
	}
	var rolloutPercentage = getCurrentRolloutPercentage(rollout, time.Now())
	return rolloutPercentage != 0 && (boundedHash <= rolloutPercentage)
}

type segmentedFeatureData struct {
	Feature ConfigFeature `json:"feature"`
	Target  Target        `json:"target"`
}

func evaluateSegmentationForFeature(config *configBody, feature ConfigFeature, user api.DVCPopulatedUser, clientCustomData map[string]interface{}) *Target {
	for _, target := range feature.Configuration.Targets {
		if _evaluateOperator(target.Audience.Filters, config.Audiences, user, clientCustomData) {
			return target
		}
	}
	return nil
}

func getSegmentedFeatureDataFromConfig(config *configBody, user api.DVCPopulatedUser, clientCustomData map[string]interface{}) []segmentedFeatureData {
	var accumulator []segmentedFeatureData
	for _, feature := range config.Features {
		segmentedFeatureTarget := evaluateSegmentationForFeature(config, feature, user, clientCustomData)

		if segmentedFeatureTarget != nil {
			featureData := segmentedFeatureData{
				Feature: feature,
				Target:  *segmentedFeatureTarget,
			}
			accumulator = append(accumulator, featureData)
		}
	}
	return accumulator
}

type targetAndHashes struct {
	Target Target
	Hashes boundedHash
}

func doesUserQualifyForFeature(config *configBody, feature ConfigFeature, user api.DVCPopulatedUser, clientCustomData map[string]interface{}) (targetAndHashes, error) {
	target := evaluateSegmentationForFeature(config, feature, user, clientCustomData)
	if target == nil {
		return targetAndHashes{}, fmt.Errorf("user %s does not qualify for any targets for feature %s", user.UserId, feature.Key)
	}

	boundedHashes := generateBoundedHashes(user.UserId, target.Id)
	rolloutHash := boundedHashes.RolloutHash

	if target.Rollout != nil && !doesUserPassRollout(*target.Rollout, rolloutHash) {
		return targetAndHashes{}, fmt.Errorf("user %s does not qualify for feature %s rollout", user.UserId, feature.Key)
	}
	return targetAndHashes{
		Target: *target,
		Hashes: boundedHashes,
	}, nil
}

func bucketUserForVariation(feature *ConfigFeature, hashes targetAndHashes) (Variation, error) {
	variationId, err := hashes.Target.DecideTargetVariation(hashes.Hashes.BucketingHash)
	if err != nil {
		return Variation{}, err
	}
	for _, v := range feature.Variations {
		if v.Id == variationId {
			return v, nil
		}
	}
	return Variation{}, fmt.Errorf("config missing variation %s", variationId)
}

func GenerateBucketedConfig(sdkKey string, user api.DVCPopulatedUser, clientCustomData map[string]interface{}) (*api.BucketedUserConfig, error) {
	config, err := getConfig(sdkKey)
	if err != nil {
		return nil, err
	}
	variableMap := make(map[string]api.ReadOnlyVariable)
	featureKeyMap := make(map[string]api.Feature)
	featureVariationMap := make(map[string]string)
	variableVariationMap := make(map[string]api.FeatureVariation)

	for _, feature := range config.Features {
		thash, err := doesUserQualifyForFeature(config, feature, user, clientCustomData)
		if err != nil {
			continue
		}

		variation, err := bucketUserForVariation(&feature, thash)
		if err != nil {
			continue
		}
		featureKeyMap[feature.Key] = api.Feature{
			Id:            feature.Id,
			Type_:         feature.Type,
			Key:           feature.Key,
			Variation:     variation.Id,
			VariationKey:  variation.Key,
			VariationName: variation.Name,
		}
		featureVariationMap[feature.Id] = variation.Id

		for _, variationVar := range variation.Variables {
			variable := config.GetVariableForId(variationVar.Var)
			if variable == nil {
				return nil, fmt.Errorf("config missing variable: %s", variationVar.Var)
			}

			variableVariationMap[variable.Key] = api.FeatureVariation{
				Variation: variation.Id,
				Feature:   feature.Id,
			}
			newVar := api.ReadOnlyVariable{
				BaseVariable: api.BaseVariable{
					Key:   variable.Key,
					Type_: variable.Type,
					Value: variationVar.Value,
				},
				Id: variable.Id,
			}
			variableMap[variable.Key] = newVar
		}
	}

	return &api.BucketedUserConfig{
		Project:              config.Project,
		Environment:          config.Environment,
		Features:             featureKeyMap,
		FeatureVariationMap:  featureVariationMap,
		VariableVariationMap: variableVariationMap,
		Variables:            variableMap,
	}, nil
}

type BucketedVariableResponse struct {
	Variable  api.ReadOnlyVariable
	Feature   ConfigFeature
	Variation Variation
}

var emptyVariableVariationMap = map[string]api.FeatureVariation{}

func VariableForUser(sdkKey string, user api.DVCPopulatedUser, variableKey string, variableType string, eventQueue *EventQueue, clientCustomData map[string]interface{}) (*api.ReadOnlyVariable, error) {
	var variablePtr *api.ReadOnlyVariable = nil
	variableVariationMap := map[string]api.FeatureVariation{}
	result, err := generateBucketedVariableForUser(sdkKey, user, variableKey, clientCustomData)
	if err != nil {
		eventErr := eventQueue.QueueVariableEvaluatedEvent(emptyVariableVariationMap, nil, variableKey)
		if eventErr != nil {
			util.Warnf("Failed to queue variable defaulted event: %s", eventErr)
		}
		return nil, err
	}

	if _, ok := VariableTypes[variableType]; !ok || result.Variable.Type_ != variableType {
		result = nil
	}

	if result != nil {
		variablePtr = &result.Variable
	}

	if !eventQueue.options.DisableAutomaticEventLogging {
		if result != nil {
			variableVariationMap[variableKey] = api.FeatureVariation{
				Variation: result.Variation.Id,
				Feature:   result.Feature.Id,
			}
		}

		err = eventQueue.QueueVariableEvaluatedEvent(variableVariationMap, variablePtr, variableKey)
		if err != nil {
			util.Warnf("Failed to queue variable evaluated event: %s", err)
		}
	}
	return variablePtr, nil

}

func generateBucketedVariableForUser(sdkKey string, user api.DVCPopulatedUser, key string, clientCustomData map[string]interface{}) (*BucketedVariableResponse, error) {
	config, err := getConfig(sdkKey)
	if err != nil {
		return nil, err
	}
	variable := config.GetVariableForKey(key)
	if variable == nil {
		return nil, fmt.Errorf("config missing variable %s", key)
	}
	featForVariable := config.GetFeatureForVariableId(variable.Id)
	if featForVariable == nil {
		return nil, fmt.Errorf("config missing feature for variable %s", key)
	}

	th, err := doesUserQualifyForFeature(config, *featForVariable, user, clientCustomData)
	if err != nil {
		return nil, err
	}
	variation, err := bucketUserForVariation(featForVariable, th)
	if err != nil {
		return nil, err
	}
	variationVariable := variation.GetVariableById(variable.Id)
	if variationVariable == nil {
		return nil, fmt.Errorf("config processing error: config missing variable %s for variation %s", key, variation.Id)
	}
	return &BucketedVariableResponse{
		Variable: api.ReadOnlyVariable{
			Id: variable.Id,
			BaseVariable: api.BaseVariable{
				Type_: variable.Type,
				Key:   variable.Key,
				Value: variationVariable.Value,
			},
		},
		Feature:   *featForVariable,
		Variation: variation,
	}, nil
}
