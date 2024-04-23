package bucketing

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/go-playground/validator/v10"

	"github.com/devcyclehq/go-server-sdk/v2/api"
)

// use a single instance of Validate, it caches struct info
var validate *validator.Validate

func init() {
	validate = validator.New()
}

type Variable struct {
	Id   string `json:"_id" validate:"required"`
	Type string `json:"type" validate:"oneof=String Boolean Number JSON"`
	Key  string `json:"key" validate:"required"`
}

type configBody struct {
	Project                api.Project             `json:"project" validate:"required"`
	Audiences              map[string]NoIdAudience `json:"audiences"`
	Environment            api.Environment         `json:"environment" validate:"required"`
	Features               []*ConfigFeature        `json:"features" validate:"required"`
	Variables              []*Variable             `json:"variables" validate:"required,dive"`
	etag                   string
	rayId                  string
	variableIdMap          map[string]*Variable
	variableKeyMap         map[string]*Variable
	variableIdToFeatureMap map[string]*ConfigFeature
}

func newConfig(configJSON []byte, etag string, rayId string) (*configBody, error) {
	config := configBody{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, err
	}
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("Config validation failed: %w", err)
	}
	if config.Audiences == nil {
		config.Audiences = make(map[string]NoIdAudience)
	}
	config.compile(etag, rayId)
	return &config, nil
}

func (c *configBody) GetVariableForKey(key string) *Variable {
	if variable, ok := c.variableKeyMap[key]; ok {
		return variable
	}
	return nil
}

func (c *configBody) GetVariableForId(id string) *Variable {
	if variable, ok := c.variableIdMap[id]; ok {
		return variable
	}
	return nil
}

func (c *configBody) GetFeatureForVariableId(id string) *ConfigFeature {
	if feature, ok := c.variableIdToFeatureMap[id]; ok {
		return feature
	}
	return nil
}

func (c *configBody) compile(etag string, rayId string) {
	// Build mappings of IDs and keys to features and variables.
	variableIdToFeatureMap := make(map[string]*ConfigFeature)
	for _, feature := range c.Features {
		for _, v := range feature.Variations {
			for _, vv := range v.Variables {
				if _, ok := variableIdToFeatureMap[vv.Var]; !ok {
					variableIdToFeatureMap[vv.Var] = feature
				}
			}
		}
	}

	variableKeyMap := make(map[string]*Variable, len(c.Variables))
	variableIdMap := make(map[string]*Variable, len(c.Variables))
	for _, variable := range c.Variables {
		variableKeyMap[variable.Key] = variable
		variableIdMap[variable.Id] = variable
	}

	c.variableIdToFeatureMap = variableIdToFeatureMap
	c.variableIdMap = variableIdMap
	c.variableKeyMap = variableKeyMap
	c.etag = etag
	c.rayId = rayId

	// Sort the feature distributions by "_variation" attribute in descending alphabetical order
	for _, feature := range c.Features {
		for _, target := range feature.Configuration.Targets {
			sort.Slice(target.Distribution, func(i, j int) bool {
				return target.Distribution[i].Variation > target.Distribution[j].Variation
			})
		}
	}
}

func (c *configBody) Equals(c2 configBody) bool {
	return reflect.DeepEqual(*c, c2)
}
