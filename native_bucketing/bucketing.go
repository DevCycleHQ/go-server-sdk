package native_bucketing

import (
	"errors"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
)

// Max value of an unsigned 32-bit integer, which is what murmurhash returns
const maxHashValue uint32 = 4294967295
const baseSeed = 1

var ErrMissingVariableForVariation = errors.New("Config missing variable for variation")
var ErrMissingFeature = errors.New("Config missing feature for variable")
var ErrMissingVariable = errors.New("Config missing variable")
var ErrMissingVariation = errors.New("Config missing variation")
var ErrUserRollout = errors.New("User does not qualify for feature rollout")
var ErrUserDoesNotQualifyForTargets = errors.New("User does not qualify for any targets for feature")
var ErrInvalidVariableType = errors.New("Invalid variable type")

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

func evaluateSegmentationForFeature(config *configBody, feature *ConfigFeature, user api.PopulatedUser, clientCustomData map[string]interface{}) *Target {
	for _, target := range feature.Configuration.Targets {
		if _evaluateOperator(target.Audience.Filters, config.Audiences, user, clientCustomData) {
			return target
		}
	}
	return nil
}

type targetAndHashes struct {
	Target Target
	Hashes boundedHash
}

func doesUserQualifyForFeature(config *configBody, feature *ConfigFeature, user api.PopulatedUser, clientCustomData map[string]interface{}) (targetAndHashes, error) {
	target := evaluateSegmentationForFeature(config, feature, user, clientCustomData)
	if target == nil {
		return targetAndHashes{}, ErrUserDoesNotQualifyForTargets
	}

	boundedHashes := generateBoundedHashes(user.UserId, target.Id)
	rolloutHash := boundedHashes.RolloutHash

	if target.Rollout != nil && !doesUserPassRollout(*target.Rollout, rolloutHash) {
		return targetAndHashes{}, ErrUserRollout
	}
	return targetAndHashes{
		Target: *target,
		Hashes: boundedHashes,
	}, nil
}

func bucketUserForVariation(feature *ConfigFeature, hashes targetAndHashes) (*Variation, error) {
	variationId, err := hashes.Target.DecideTargetVariation(hashes.Hashes.BucketingHash)
	if err != nil {
		return nil, err
	}
	for _, v := range feature.Variations {
		if v.Id == variationId {
			return v, nil
		}
	}
	return nil, ErrMissingVariation
}

func GenerateBucketedConfig(sdkKey string, user api.PopulatedUser, clientCustomData map[string]interface{}) (*api.BucketedUserConfig, error) {
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

		variation, err := bucketUserForVariation(feature, thash)
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
				return nil, ErrMissingVariable
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

func VariableForUser(sdkKey string, user api.PopulatedUser, variableKey string, expectedVariableType string, eventQueue *EventQueue, clientCustomData map[string]interface{}) (variableType string, variableValue any, err error) {
	variableType, variableValue, featureId, variationId, err := generateBucketedVariableForUser(sdkKey, user, variableKey, clientCustomData)
	if err != nil {
		// TODO: convert this flow to just handle variable defaulted
		eventErr := eventQueue.QueueVariableEvaluatedEvent(variableKey, "", "", true)
		if eventErr != nil {
			util.Warnf("Failed to queue variable defaulted event: %s", eventErr)
		}
		return "", nil, err
	}

	var variableDefaulted bool
	if !isVariableTypeValid(variableType, expectedVariableType) {
		err = ErrInvalidVariableType
		variableDefaulted = true
	}

	if !eventQueue.options.DisableAutomaticEventLogging {
		eventErr := eventQueue.QueueVariableEvaluatedEvent(variableKey, featureId, variationId, variableDefaulted)
		if eventErr != nil {
			util.Warnf("Failed to queue variable evaluated event: %s", eventErr)
		}
	}

	return
}

func isVariableTypeValid(variableType string, expectedVariableType string) bool {
	if variableType != VariableTypesString &&
		variableType != VariableTypesNumber &&
		variableType != VariableTypesJSON &&
		variableType != VariableTypesBool {
		return false
	}
	if variableType != expectedVariableType {
		return false
	}
	return true
}

func generateBucketedVariableForUser(sdkKey string, user api.PopulatedUser, key string, clientCustomData map[string]interface{}) (variableType string, variableValue any, featureId string, variationId string, err error) {
	config, err := getConfig(sdkKey)
	if err != nil {
		return
	}
	variable := config.GetVariableForKey(key)
	if variable == nil {
		err = ErrMissingVariable
		return
	}
	featForVariable := config.GetFeatureForVariableId(variable.Id)
	if featForVariable == nil {
		err = ErrMissingFeature
		return
	}

	th, err := doesUserQualifyForFeature(config, featForVariable, user, clientCustomData)
	if err != nil {
		return
	}
	variation, err := bucketUserForVariation(featForVariable, th)
	if err != nil {
		return
	}
	variationVariable := variation.GetVariableById(variable.Id)
	if variationVariable == nil {
		err = ErrMissingVariableForVariation
		return
	}
	return variable.Type, variationVariable.Value, featForVariable.Id, variation.Id, nil
}
