package devcycle

import (
	"context"
	"errors"

	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/open-feature/go-sdk/pkg/openfeature"
)

// DevCycleProvider implements the FeatureProvider interface and provides functions for evaluating flags
type DevCycleProvider struct {
	Client ClientImpl
}

type ClientImpl interface {
	Variable(userdata User, key string, defaultValue interface{}) (Variable, error)
	IsLocalBucketing() bool
}

// Metadata returns the metadata of the provider
func (p DevCycleProvider) Metadata() openfeature.Metadata {
	if p.Client.IsLocalBucketing() {
		return openfeature.Metadata{Name: "DevCycleProvider Local"}
	} else {
		return openfeature.Metadata{Name: "DevCycleProvider Cloud"}
	}
}

// Convenience method for creating a DevCycleProvider from a Client
func (c *Client) OpenFeatureProvider() DevCycleProvider {
	return DevCycleProvider{Client: c}
}

// BooleanEvaluation returns a boolean flag
func (p DevCycleProvider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	user, err := createUserFromEvaluationContext(evalCtx)
	if err != nil {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewInvalidContextResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	variable, err := p.Client.Variable(user, flag, defaultValue)

	if err != nil {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: toOpenFeatureError(err), Reason: openfeature.ErrorReason,
			},
		}
	}

	if variable.IsDefaulted {
		return openfeature.BoolResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason},
		}
	}

	switch variable.Value.(type) {
	case bool:
		return openfeature.BoolResolutionDetail{
			Value:                    variable.Value.(bool),
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.TargetingMatchReason},
		}
	case nil:
		return openfeature.BoolResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason},
		}
	default:
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewTypeMismatchResolutionError("Variable result is nil"),
			},
		}
	}
}

// StringEvaluation returns a string flag
func (p DevCycleProvider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	user, err := createUserFromEvaluationContext(evalCtx)
	if err != nil {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewInvalidContextResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	variable, err := p.Client.Variable(user, flag, defaultValue)
	if err != nil {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: toOpenFeatureError(err), Reason: openfeature.ErrorReason,
			},
		}
	}

	if variable.IsDefaulted {
		return openfeature.StringResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason},
		}
	}

	switch variable.Value.(type) {
	case string:
		return openfeature.StringResolutionDetail{
			Value:                    variable.Value.(string),
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.TargetingMatchReason},
		}
	case nil:
		return openfeature.StringResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason},
		}
	default:
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewTypeMismatchResolutionError("Variable result is nil"),
			},
		}
	}
}

// FloatEvaluation returns a float flag
func (p DevCycleProvider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	user, err := createUserFromEvaluationContext(evalCtx)
	if err != nil {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewInvalidContextResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	variable, err := p.Client.Variable(user, flag, defaultValue)
	if err != nil {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: toOpenFeatureError(err), Reason: openfeature.ErrorReason,
			},
		}
	}

	if variable.IsDefaulted {
		return openfeature.FloatResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason},
		}
	}

	switch castValue := variable.Value.(type) {
	case float64:
		return openfeature.FloatResolutionDetail{
			Value:                    castValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.TargetingMatchReason},
		}
	case nil:
		return openfeature.FloatResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason},
		}
	default:
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewTypeMismatchResolutionError("Variable result is nil"),
			},
		}
	}
}

// IntEvaluation returns an int flag
func (p DevCycleProvider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	user, err := createUserFromEvaluationContext(evalCtx)
	if err != nil {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewInvalidContextResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	variable, err := p.Client.Variable(user, flag, defaultValue)
	if err != nil {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: toOpenFeatureError(err), Reason: openfeature.ErrorReason,
			},
		}
	}

	if variable.IsDefaulted {
		return openfeature.IntResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason},
		}
	}

	switch castValue := variable.Value.(type) {
	case float64:
		return openfeature.IntResolutionDetail{
			Value:                    int64(castValue),
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.TargetingMatchReason},
		}
	case nil:
		return openfeature.IntResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason},
		}
	default:
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewTypeMismatchResolutionError("Variable result is nil"),
			},
		}
	}
}

// ObjectEvaluation returns an object flag
func (p DevCycleProvider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {

	user, err := createUserFromEvaluationContext(evalCtx)
	if err != nil {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewInvalidContextResolutionError(err.Error()), Reason: openfeature.ErrorReason,
			},
		}
	}

	variable, err := p.Client.Variable(user, flag, defaultValue)
	if err != nil {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: toOpenFeatureError(err), Reason: openfeature.ErrorReason,
			},
		}
	}

	if variable.IsDefaulted || variable.Value == nil {
		return openfeature.InterfaceResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.DefaultReason},
		}
	}

	return openfeature.InterfaceResolutionDetail{
		Value:                    variable.Value,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.TargetingMatchReason},
	}
}

// Hooks returns hooks
func (p DevCycleProvider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

func toOpenFeatureError(err error) openfeature.ResolutionError {
	if errors.Is(err, ErrInvalidDefaultValue) {
		return openfeature.NewTypeMismatchResolutionError(err.Error())
	}
	return openfeature.NewGeneralResolutionError(err.Error())
}

func createUserFromEvaluationContext(evalCtx openfeature.FlattenedContext) (User, error) {
	userId := ""
	_, exists := evalCtx[openfeature.TargetingKey]
	if exists {
		userId = evalCtx[openfeature.TargetingKey].(string)
	} else {
		_, exists = evalCtx["userId"]
		if exists {
			userId = evalCtx["userId"].(string)
		}
	}

	if userId == "" {
		return DVCUser{}, errors.New("targetingKey or userId must be provided")
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
