package devcycle

import (
	"context"
	"github.com/open-feature/go-sdk/pkg/openfeature"
)

// DevCycleProvider implements the FeatureProvider interface and provides functions for evaluating flags
type DevCycleProvider struct {
	Client *DVCClient
}

// Metadata returns the metadata of the provider
func (p DevCycleProvider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{Name: "DevCycleProvider"}
}

func createUserFromFeatureContext(evalCtx openfeature.FlattenedContext) (DVCUser, error) {
	// need some actual error handling around the construction of the user
	user := DVCUser{
		UserId:            evalCtx["userId"].(string),
		Email:             evalCtx["email"].(string),
		Name:              evalCtx["name"].(string),
		Language:          evalCtx["language"].(string),
		Country:           evalCtx["country"].(string),
		AppVersion:        evalCtx["appVersion"].(string),
		AppBuild:          evalCtx["appBuild"].(string),
		CustomData:        evalCtx["customData"].(map[string]interface{}),
		PrivateCustomData: evalCtx["privateCustomData"].(map[string]interface{}),
		DeviceModel:       evalCtx["deviceModel"].(string),
	}

	return user, nil
}

// BooleanEvaluation returns a boolean flag
func (p DevCycleProvider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {

	user, err := createUserFromFeatureContext(evalCtx)
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

	return openfeature.BoolResolutionDetail{Value: variable.Value.(bool)}
}

// StringEvaluation returns a string flag
func (p DevCycleProvider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	user, err := createUserFromFeatureContext(evalCtx)
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

	return openfeature.StringResolutionDetail{Value: variable.Value.(string)}
}

// FloatEvaluation returns a float flag
func (p DevCycleProvider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	user, err := createUserFromFeatureContext(evalCtx)
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

	return openfeature.FloatResolutionDetail{Value: variable.Value.(float64)}
}

// IntEvaluation returns an int flag
func (p DevCycleProvider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	user, err := createUserFromFeatureContext(evalCtx)
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

	return openfeature.IntResolutionDetail{Value: variable.Value.(int64)}
}

// ObjectEvaluation returns an object flag
func (p DevCycleProvider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {

	user, err := createUserFromFeatureContext(evalCtx)
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

	return openfeature.InterfaceResolutionDetail{Value: variable.Value}
}

// Hooks returns hooks
func (p DevCycleProvider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}
