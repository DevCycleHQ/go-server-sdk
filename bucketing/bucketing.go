package bucketing

import (
	"errors"
	"strconv"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
)

// Max value of an unsigned 32-bit integer, which is what murmurhash returns
const maxHashValue uint32 = 4294967295
const baseSeed = 1
const defaultBucketingValue = "null"

var ErrMissingVariableForVariation = errors.New("Config missing variable for variation")
var ErrMissingFeature = errors.New("Config missing feature for variable")
var ErrMissingVariable = errors.New("Config missing variable")
var ErrMissingVariation = errors.New("Config missing variation")
var ErrFailedToDecideVariation = errors.New("Failed to decide target variation")
var ErrUserRollout = errors.New("User does not qualify for feature rollout")
var ErrUserDoesNotQualifyForTargets = errors.New("User does not qualify for any targets for feature")
var ErrInvalidVariableType = errors.New("Invalid variable type")
var ErrConfigMissing = errors.New("No config available")

type boundedHash struct {
	RolloutHash   float64 `json:"rolloutHash"`
	BucketingHash float64 `json:"bucketingHash"`
}

func generateBoundedHashes(bucketingKeyValue, targetId string) boundedHash {
	var targetHash = murmurhashV3(targetId, baseSeed)
	var bhash = boundedHash{
		RolloutHash:   generateBoundedHash(bucketingKeyValue+"_rollout", targetHash),
		BucketingHash: generateBoundedHash(bucketingKeyValue, targetHash),
	}
	return bhash
}

func generateBoundedHash(input string, hashSeed uint32) float64 {
	mh := murmurhashV3(input, hashSeed)
	return float64(mh) / float64(maxHashValue)
}

func determineUserBucketingValueForTarget(targetBucketingKey, userId string, mergedCustomData map[string]interface{}) string {
	if targetBucketingKey == "" || targetBucketingKey == "user_id" {
		return userId
	}

	if customDataValue, keyExists := mergedCustomData[targetBucketingKey]; keyExists {
		if customDataValue == nil {
			return defaultBucketingValue
		}

		switch v := customDataValue.(type) {
		case int:
			return strconv.Itoa(v)
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case string:
			return v
		case bool:
			return strconv.FormatBool(v)
		default:
			return defaultBucketingValue
		}
	}
	return defaultBucketingValue
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
	var rolloutPercentage = getCurrentRolloutPercentage(rollout, time.Now())
	return rolloutPercentage != 0 && (boundedHash <= rolloutPercentage)
}

func evaluateSegmentationForFeature(config *configBody, feature *ConfigFeature, user api.PopulatedUser, clientCustomData map[string]interface{}) *Target {
	var mergedCustomData = user.CombinedCustomData()
	for _, target := range feature.Configuration.Targets {
		passthroughEnabled := !config.Project.Settings.DisablePassthroughRollouts
		doesUserPassthrough := true
		if target.Rollout != nil && passthroughEnabled {
			var bucketingValue = determineUserBucketingValueForTarget(target.BucketingKey, user.UserId, mergedCustomData)

			boundedHash := generateBoundedHashes(bucketingValue, target.Id)
			rolloutHash := boundedHash.RolloutHash
			doesUserPassthrough = doesUserPassRollout(*target.Rollout, rolloutHash)
		}
		operator := target.Audience.Filters
		if doesUserPassthrough && operator.Evaluate(config.Audiences, user, clientCustomData) {
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

	var mergedCustomData = user.CombinedCustomData()
	var bucketingValue = determineUserBucketingValueForTarget(target.BucketingKey, user.UserId, mergedCustomData)

	boundedHashes := generateBoundedHashes(bucketingValue, target.Id)
	rolloutHash := boundedHashes.RolloutHash
	passthroughEnabled := !config.Project.Settings.DisablePassthroughRollouts

	if target.Rollout != nil && !passthroughEnabled && !doesUserPassRollout(*target.Rollout, rolloutHash) {
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
			return nil, err
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
		eventErr := eventQueue.QueueVariableDefaultedEvent(variableKey, BucketResultErrorToDefaultReason(err))
		if eventErr != nil {
			util.Warnf("Failed to queue variable defaulted event: %s", eventErr)
		}
		return "", nil, err
	}

	if !isVariableTypeValid(variableType, expectedVariableType) && expectedVariableType != "" {
		err = ErrInvalidVariableType
		eventErr := eventQueue.QueueVariableDefaultedEvent(variableKey, BucketResultErrorToDefaultReason(err))
		if eventErr != nil {
			util.Warnf("Failed to queue variable defaulted event: %s", eventErr)
		}
		return "", nil, err
	}

	eventErr := eventQueue.QueueVariableEvaluatedEvent(variableKey, featureId, variationId)
	if eventErr != nil {
		util.Warnf("Failed to queue variable evaluated event: %s", eventErr)
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
		util.Warnf("Variable called before client initialized, returning default value")
		return "", nil, "", "", ErrConfigMissing
	}
	variable := config.GetVariableForKey(key)
	if variable == nil {
		err = ErrMissingVariable
		return "", nil, "", "", err
	}
	featForVariable := config.GetFeatureForVariableId(variable.Id)
	if featForVariable == nil {
		err = ErrMissingFeature
		return "", nil, "", "", err
	}

	th, err := doesUserQualifyForFeature(config, featForVariable, user, clientCustomData)
	if err != nil {
		return "", nil, "", "", err
	}
	variation, err := bucketUserForVariation(featForVariable, th)
	if err != nil {
		return "", nil, "", "", err
	}
	variationVariable := variation.GetVariableById(variable.Id)
	if variationVariable == nil {
		err = ErrMissingVariableForVariation
		return "", nil, "", "", err
	}
	return variable.Type, variationVariable.Value, featForVariable.Id, variation.Id, nil
}

func BucketResultErrorToDefaultReason(err error) (defaultReason string) {
	switch err {
	case ErrConfigMissing:
		return "MISSING_CONFIG"
	case ErrMissingVariable:
		return "MISSING_VARIABLE"
	case ErrMissingFeature:
		return "MISSING_FEATURE"
	case ErrMissingVariation:
		return "MISSING_VARIATION"
	case ErrMissingVariableForVariation:
		return "MISSING_VARIABLE_FOR_VARIATION"
	case ErrUserRollout:
		return "USER_NOT_IN_ROLLOUT"
	case ErrUserDoesNotQualifyForTargets:
		return "USER_NOT_TARGETED"
	case ErrInvalidVariableType:
		return "INVALID_VARIABLE_TYPE"
	default:
		return "Unknown"
	}
}
