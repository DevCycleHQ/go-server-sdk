package bucketing

import (
	"errors"
	"strconv"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
)

var allEvalReasons = []api.EvaluationReason{
	api.EvaluationReasonTargetingMatch,
	api.EvaluationReasonSplit,
	api.EvaluationReasonDefault,
	api.EvaluationReasonError,
}

// Max value of an unsigned 32-bit integer, which is what murmurhash returns
const maxHashValue uint32 = 4294967295
const baseSeed = 1
const defaultBucketingValue = "null"

var ErrMissingVariableForVariation = errors.New("config missing variable for variation")
var ErrMissingFeature = errors.New("config missing feature for variable")
var ErrMissingVariable = errors.New("config missing variable")
var ErrMissingVariation = errors.New("config missing variation")
var ErrFailedToDecideVariation = errors.New("failed to decide target variation")
var ErrUserRollout = errors.New("user does not qualify for feature rollout")
var ErrUserDoesNotQualifyForTargets = errors.New("user does not qualify for any targets for feature")
var ErrInvalidVariableType = errors.New("invalid variable type")
var ErrConfigMissing = errors.New("no config available")

type boundedHashType struct {
	RolloutHash   float64 `json:"rolloutHash"`
	BucketingHash float64 `json:"bucketingHash"`
}

func generateBoundedHashes(bucketingKeyValue, targetId string) boundedHashType {
	var targetHash = murmurhashV3(targetId, baseSeed)
	var bhash = boundedHashType{
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

func isUserInRollout(rollout Rollout, boundedHash float64) bool {
	var rolloutPercentage = getCurrentRolloutPercentage(rollout, time.Now())
	return rolloutPercentage != 0 && (boundedHash <= rolloutPercentage)
}

func evaluateSegmentationForFeature(config *configBody, feature *ConfigFeature, user api.PopulatedUser, clientCustomData map[string]interface{}) (t *Target, isRollout bool) {
	var mergedCustomData = user.CombinedCustomData()
	for _, target := range feature.Configuration.Targets {
		passthroughEnabled := !config.Project.Settings.DisablePassthroughRollouts
		rolloutCriteriaMet := true
		if target.Rollout != nil && passthroughEnabled {

			var bucketingValue = determineUserBucketingValueForTarget(target.BucketingKey, user.UserId, mergedCustomData)

			boundedHash := generateBoundedHashes(bucketingValue, target.Id)
			rolloutHash := boundedHash.RolloutHash
			rolloutCriteriaMet = isUserInRollout(*target.Rollout, rolloutHash)
			isRollout = rolloutCriteriaMet
		}
		operator := target.Audience.Filters
		if rolloutCriteriaMet && operator.Evaluate(config.Audiences, user, clientCustomData) {
			return target, isRollout
		}
	}
	return nil, false
}

type targetAndHashes struct {
	Target Target
	Hashes boundedHashType
}

func doesUserQualifyForFeature(config *configBody, feature *ConfigFeature, user api.PopulatedUser, clientCustomData map[string]interface{}) (targetAndHashes, bool, error) {
	target, isRollout := evaluateSegmentationForFeature(config, feature, user, clientCustomData)
	if target == nil {
		return targetAndHashes{}, isRollout, ErrUserDoesNotQualifyForTargets
	}

	var mergedCustomData = user.CombinedCustomData()
	var bucketingValue = determineUserBucketingValueForTarget(target.BucketingKey, user.UserId, mergedCustomData)

	boundedHashes := generateBoundedHashes(bucketingValue, target.Id)
	rolloutHash := boundedHashes.RolloutHash
	passthroughEnabled := !config.Project.Settings.DisablePassthroughRollouts

	if target.Rollout != nil && !passthroughEnabled && !isUserInRollout(*target.Rollout, rolloutHash) {
		return targetAndHashes{}, true, ErrUserRollout
	}
	return targetAndHashes{
		Target: *target,
		Hashes: boundedHashes,
	}, isRollout, nil
}

func bucketUserForVariation(feature *ConfigFeature, hashes targetAndHashes) (*Variation, bool, error) {
	variationId, isRandomDistrib, err := hashes.Target.DecideTargetVariation(hashes.Hashes.BucketingHash)
	if err != nil {
		return nil, isRandomDistrib, err
	}
	for _, v := range feature.Variations {
		if v.Id == variationId {
			return v, isRandomDistrib, nil
		}
	}
	return nil, isRandomDistrib, ErrMissingVariation
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
		thash, _, err := doesUserQualifyForFeature(config, feature, user, clientCustomData)
		if err != nil {
			continue
		}

		variation, _, err := bucketUserForVariation(feature, thash)
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
					Eval: api.EvalDetails{
						Reason:  api.EvaluationReasonTargetingMatch,
						Details: "",
					},
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

func VariableForUser(sdkKey string, user api.PopulatedUser, variableKey string, expectedVariableType string, eventQueue *EventQueue, clientCustomData map[string]interface{}) (variableType string, variableValue any, evalReason api.EvaluationReason, evalDetails string, err error) {
	variableType, variableValue, featureId, variationId, evalReason, err := generateBucketedVariableForUser(sdkKey, user, variableKey, clientCustomData)
	if err != nil {
		eventErr := eventQueue.QueueVariableDefaultedEvent(variableKey, BucketResultErrorToDefaultReason(err))
		if eventErr != nil {
			util.Warnf("Failed to queue variable defaulted event: %s", eventErr)
		}
		return "", nil, evalReason, string(BucketResultErrorToDefaultReason(err)), err
	}

	if !isVariableTypeValid(variableType, expectedVariableType) && expectedVariableType != "" {
		err = ErrInvalidVariableType
		eventErr := eventQueue.QueueVariableDefaultedEvent(variableKey, BucketResultErrorToDefaultReason(err))
		if eventErr != nil {
			util.Warnf("Failed to queue variable defaulted event: %s", eventErr)
		}
		return "", nil, evalReason, string(BucketResultErrorToDefaultReason(err)), err
	}

	eventErr := eventQueue.QueueVariableEvaluatedEvent(variableKey, featureId, variationId, evalReason)
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

func generateBucketedVariableForUser(sdkKey string, user api.PopulatedUser, key string, clientCustomData map[string]interface{}) (variableType string, variableValue any, featureId string, variationId string, evalReason api.EvaluationReason, err error) {
	config, err := getConfig(sdkKey)
	if err != nil {
		util.Warnf("Variable called before client initialized, returning default value")
		return "", nil, "", "", api.EvaluationReasonError, ErrConfigMissing
	}
	variable := config.GetVariableForKey(key)
	if variable == nil {
		err = ErrMissingVariable
		return "", nil, "", "", api.EvaluationReasonDisabled, err
	}
	featForVariable := config.GetFeatureForVariableId(variable.Id)
	if featForVariable == nil {
		err = ErrMissingFeature
		return "", nil, "", "", api.EvaluationReasonDisabled, err
	}

	targetHashes, isRollout, err := doesUserQualifyForFeature(config, featForVariable, user, clientCustomData)
	if err != nil {
		return "", nil, "", "", api.EvaluationReasonDefault, err
	}
	variation, isRandomDistrib, err := bucketUserForVariation(featForVariable, targetHashes)
	if err != nil {
		return "", nil, "", "", api.EvaluationReasonDefault, err
	}
	variationVariable := variation.GetVariableById(variable.Id)
	if variationVariable == nil {
		err = ErrMissingVariableForVariation
		return "", nil, "", "", api.EvaluationReasonDisabled, err
	}
	if isRollout || isRandomDistrib {
		return variable.Type, variationVariable.Value, featForVariable.Id, variation.Id, api.EvaluationReasonSplit, nil
	}
	return variable.Type, variationVariable.Value, featForVariable.Id, variation.Id, api.EvaluationReasonTargetingMatch, nil
}

func BucketResultErrorToDefaultReason(err error) (defaultReason api.DefaultReason) {
	switch err {
	case ErrConfigMissing:
		return api.DefaultReasonMissingConfig
	case ErrMissingVariable:
		return api.DefaultReasonMissingVariable
	case ErrMissingFeature:
		return api.DefaultReasonMissingFeature
	case ErrMissingVariation:
		return api.DefaultReasonMissingVariation
	case ErrMissingVariableForVariation:
		return api.DefaultReasonMissingVariableForVariation
	case ErrUserRollout:
		return api.DefaultReasonUserNotInRollout
	case ErrUserDoesNotQualifyForTargets:
		return api.DefaultReasonUserNotTargeted
	case ErrInvalidVariableType:
		return api.DefaultReasonInvalidVariableType
	case nil:
		return api.DefaultReasonNotDefaulted
	default:
		return api.DefaultReasonUnknown
	}
}
