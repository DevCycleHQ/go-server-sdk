package native_bucketing

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Variable struct {
	Id   string `json:"_id"`
	Type string `json:"type" validate:"regexp=^(String|Boolean|Number|JSON)$"`
	Key  string `json:"key"`
}

type configBody struct {
	Project                PublicProject           `json:"project" validate:"required"`
	Audiences              map[string]NoIdAudience `json:"audiences" validate:"required"`
	Environment            PublicEnvironment       `json:"environment" validate:"required"`
	Features               []ConfigFeature         `json:"features" validate:"required"`
	Variables              []Variable              `json:"variables" validate:"required"`
	etag                   string
	variableIdMap          map[string]Variable
	variableKeyMap         map[string]Variable
	variableIdToFeatureMap map[string]ConfigFeature
}

func (c *configBody) GetVariableForKey(key string) *Variable {
	if variable, ok := c.variableKeyMap[key]; ok {
		return &variable
	}
	return nil
}

func (c *configBody) GetVariableForId(id string) *Variable {
	if variable, ok := c.variableIdMap[id]; ok {
		return &variable
	}
	return nil
}

func (c *configBody) GetFeatureForVariableId(id string) *ConfigFeature {
	if feature, ok := c.variableIdToFeatureMap[id]; ok {
		return &feature
	}
	return nil
}

func (c *configBody) compile(etag string) {

	variableIdToFeatureMap := make(map[string]ConfigFeature)
	for _, feature := range c.Features {
		for _, v := range feature.Variations {
			for _, vv := range v.Variables {
				if _, ok := variableIdToFeatureMap[vv.Var]; !ok {
					variableIdToFeatureMap[vv.Var] = feature
				}
			}
		}
	}

	variableKeyMap := make(map[string]Variable, len(c.Variables))
	variableIdMap := make(map[string]Variable, len(c.Variables))
	for _, variable := range c.Variables {
		variableKeyMap[variable.Key] = variable
		variableIdMap[variable.Id] = variable
	}

	c.variableIdToFeatureMap = variableIdToFeatureMap
	c.variableIdMap = variableIdMap
	c.variableKeyMap = variableKeyMap
	c.etag = etag
}

func (c *configBody) FindVariable(key string) (Variable, error) {
	for _, v := range c.Variables {
		if v.Key == key {
			return v, nil
		}
	}
	return Variable{}, fmt.Errorf("variable key not found")
}

func (c *configBody) Equals(c2 configBody) bool {
	return reflect.DeepEqual(*c, c2)
}

func newConfig(configJSON []byte, etag string) (configBody, error) {
	// TODO: Replace with a proper validator.
	config := configBody{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return config, err
	}
	config.compile(etag)
	return config, nil
}
