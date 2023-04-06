package native_bucketing

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type PublicProject struct {
	Id             string                 `json:"_id"`
	Key            string                 `json:"key"`
	A0Organization string                 `json:"a0_organization"`
	Settings       map[string]interface{} `json:"settings"`
}

type PublicEnvironment struct {
	Id  string `json:"_id"`
	Key string `json:"key"`
}

type Variable struct {
	Id   string `json:"_id"`
	Type string `json:"type" validate:"regexp=^(String|Boolean|Number|JSON)$"`
	Key  string `json:"key"`
}
type ConfigBody struct {
	Project                PublicProject           `json:"project"`
	Audiences              map[string]NoIdAudience `json:"audiences"`
	Environment            PublicEnvironment       `json:"environment"`
	Features               []Feature               `json:"features"`
	Variables              []Variable              `json:"variables"`
	etag                   string
	variableIdMap          map[string]Variable
	variableKeyMap         map[string]Variable
	variableIdToFeatureMap map[string]Feature
}

func (c *ConfigBody) GetVariableForId(id string) *Variable {
	for _, v := range c.variableIdMap {
		if id == v.Id {
			return &v
		}
	}
	return nil
}

func NewConfig(configJSON []byte, etag string) (ConfigBody, error) {
	// TODO: Replace with a proper validator.
	config := ConfigBody{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return config, err
	}

	variableIdToFeatureMap := make(map[string]Feature)
	for _, feature := range config.Features {
		for _, v := range feature.Variations {
			for _, vv := range v.Variables {
				if _, ok := variableIdToFeatureMap[vv.Var]; !ok {
					variableIdToFeatureMap[vv.Var] = feature
				}
			}
		}
	}

	variableKeyMap := make(map[string]Variable, len(config.Variables))
	variableIdMap := make(map[string]Variable, len(config.Variables))
	for _, variable := range config.Variables {
		variableKeyMap[variable.Key] = variable
		variableIdMap[variable.Id] = variable
	}

	config.variableIdToFeatureMap = variableIdToFeatureMap
	config.variableIdMap = variableIdMap
	config.variableKeyMap = variableKeyMap

	return config, nil
}

func (c *ConfigBody) FindVariable(key string) (Variable, error) {
	for _, v := range c.Variables {
		if v.Key == key {
			return v, nil
		}
	}
	return Variable{}, fmt.Errorf("variable key not found")
}

func (c *ConfigBody) Equals(c2 ConfigBody) bool {
	return reflect.DeepEqual(*c, c2)
}
