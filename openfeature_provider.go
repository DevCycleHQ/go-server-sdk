package devcycle

import (
	"context"
	"errors"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/open-feature/go-sdk/pkg/openfeature"
)

// DevCycleProvider implements the FeatureProvider interface and provides functions for evaluating flags
type DevCycleProvider struct {
	Client *Client
}

// Metadata returns the metadata of the provider
func (p DevCycleProvider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{Name: "devcycle-go-provider"}
}

// BooleanEvaluation returns a boolean flag
func (p DevCycleProvider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {

	user, err := createUserFromEvaluationContext(evalCtx)
	if err != nil {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	variable, err := p.Client.Variable(user, flag, defaultValue)

	if err != nil {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	if variable.IsDefaulted {
		return openfeature.BoolResolutionDetail{Value: defaultValue, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason}}
	}

	return openfeature.BoolResolutionDetail{Value: variable.Value.(bool), ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.TargetingMatchReason}}
}

// StringEvaluation returns a string flag
func (p DevCycleProvider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	user, err := createUserFromEvaluationContext(evalCtx)
	if err != nil {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	variable, err := p.Client.Variable(user, flag, defaultValue)
	if err != nil {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	if variable.IsDefaulted {
		return openfeature.StringResolutionDetail{Value: defaultValue, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason}}
	}

	return openfeature.StringResolutionDetail{Value: variable.Value.(string), ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.TargetingMatchReason}}
}

// FloatEvaluation returns a float flag
func (p DevCycleProvider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	user, err := createUserFromEvaluationContext(evalCtx)
	if err != nil {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	variable, err := p.Client.Variable(user, flag, defaultValue)
	if err != nil {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	if variable.IsDefaulted {
		return openfeature.FloatResolutionDetail{Value: defaultValue, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason}}
	}

	return openfeature.FloatResolutionDetail{Value: variable.Value.(float64), ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.TargetingMatchReason}}
}

// IntEvaluation returns an int flag
func (p DevCycleProvider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	user, err := createUserFromEvaluationContext(evalCtx)
	if err != nil {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	variable, err := p.Client.Variable(user, flag, defaultValue)
	if err != nil {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	if variable.IsDefaulted {
		return openfeature.IntResolutionDetail{Value: defaultValue, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason}}
	}

	return openfeature.IntResolutionDetail{Value: variable.Value.(int64), ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.TargetingMatchReason}}
}

// ObjectEvaluation returns an object flag
func (p DevCycleProvider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {

	user, err := createUserFromEvaluationContext(evalCtx)
	if err != nil {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	variable, err := p.Client.Variable(user, flag, defaultValue)
	if err != nil {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	if variable.IsDefaulted {
		return openfeature.InterfaceResolutionDetail{Value: defaultValue, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason}}
	}

	return openfeature.InterfaceResolutionDetail{Value: variable.Value, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.TargetingMatchReason}}
}

// Hooks returns hooks
func (p DevCycleProvider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

func createUserFromEvaluationContext(evalCtx openfeature.FlattenedContext) (User, error) {
	userId := ""
	_, exists := evalCtx["userId"]
	if exists {
		userId = evalCtx["userId"].(string)
	} else {
		_, exists = evalCtx["targetingKey"]
		if exists {
			userId = evalCtx["targetingKey"].(string)
		}
	}

	if userId == "" {
		return DVCUser{}, errors.New("userId or targetingKey must be provided")
	}
	user := User{
		UserId: userId,
	}

	customData := make(map[string]interface{})
	privateCustomData := make(map[string]interface{})

	for key, value := range evalCtx {
		if value == nil {
			continue
		}
		if str, ok := value.(string); ok {
			if key == "email" {
				user.Email = str
			} else if key == "name" {
				user.Name = str
			} else if key == "language" {
				user.Language = str
			} else if key == "country" {
				user.Country = str
			} else if key == "appVersion" {
				user.AppVersion = str
			} else if key == "appBuild" {
				user.AppBuild = str
			} else if key == "deviceModel" {
				user.DeviceModel = str
			}
		} else if kvp, ok := value.(map[string]interface{}); ok {
			if key == "customData" {
				for k, v := range kvp {
					setCustomDataValue(customData, k, v)
				}
			} else if key == "privateCustomData" {
				for k, v := range kvp {
					setCustomDataValue(privateCustomData, k, v)
				}
			}
		} else {
			setCustomDataValue(customData, key, value)
		}
	}

	if len(customData) > 0 {
		user.CustomData = customData
	}

	if len(privateCustomData) > 0 {
		user.PrivateCustomData = privateCustomData
	}

	return user, nil
}

func setCustomDataValue(customData map[string]interface{}, key string, val interface{}) {
	if val == nil {
		return
	}
	// Custom Data only supports specific types, load the ones we can and
	// ignore the rest with warnings
	switch v := val.(type) {
	case string:
		customData[key] = v
	case float64:
		customData[key] = v
	case int:
		customData[key] = float64(v)
	case float32:
		customData[key] = float64(v)
	case int32:
		customData[key] = float64(v)
	case int64:
		customData[key] = float64(v)
	case uint:
		customData[key] = float64(v)
	case uint64:
		customData[key] = float64(v)
	case bool:
		customData[key] = v
	default:
		util.Warnf("Unsupported type for custom data value: %s=%v", key, val)
	}
}
