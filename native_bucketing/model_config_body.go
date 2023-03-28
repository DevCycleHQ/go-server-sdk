package native_bucketing

import (
	"encoding/json"
	"fmt"
	"gopkg.in/validator.v2"
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
	Project        PublicProject           `json:"project"`
	Audience       map[string]NoIdAudience `json:"audiences"`
	Environment    PublicEnvironment       `json:"environment"`
	Features       []Feature               `json:"features"`
	Variables      []Variable              `json:"variables"`
	VariableHashes map[string]int64        `json:"variableHashes"`
}

func NewConfigBody(configJSONObj map[string]interface{}, etag *string) ConfigBody {
	var audiencesJSON interface{}
	if val, ok := configJSONObj["audiences"]; ok {
		audiencesJSON = val
	} else {
		audiencesJSON = nil
	}
	var audiences map[string]NoIdAudience
	if audiencesJSON != nil {
		audiences = make(map[string]NoIdAudience)
		for audience_id, aud := range audiencesJSON.(map[string]interface{}) {
			audiences[audience_id] = NewNoIdAudience(aud.(map[string]interface{}))
		}
	} else {
		audiences = nil
	}
	featuresJSON := configJSONObj["features"].([]interface{})
	features := make([]Feature, len(featuresJSON))
	variableIdToFeatureMap := make(map[string]Feature)
	for i, f := range featuresJSON {
		feature := NewFeature(f.(map[string]interface{}))
		features[i] = feature
		for _, v := range feature.Variations {
			for _, vv := range v.Variables {
				if _, ok := variableIdToFeatureMap[vv.ID]; !ok {
					variableIdToFeatureMap[vv.ID] = feature
				}
			}
		}
	}
	variablesJSON := configJSONObj["variables"].([]interface{})
	variables := make([]Variable, len(variablesJSON))
	variableKeyMap := make(map[string]Variable)
	variableIdMap := make(map[string]Variable)
	for i, v := range variablesJSON {
		variable := NewVariable(v.(map[string]interface{}))
		variables[i] = variable
		variableKeyMap[variable.Key] = variable
		variableIdMap[variable.ID] = variable
	}
	project := NewPublicProject(configJSONObj["project"].(map[string]interface{}))
	environment := NewPublicEnvironment(configJSONObj["environment"].(map[string]interface{}))
	return ConfigBody{
		project,
		audiences,
		environment,
		features,
		variables,
		etag,
		variableKeyMap,
		variableIdMap,
		variableIdToFeatureMap,
	}
}

func (c *ConfigBody) FindVariable(key string) (error, Variable) {
	for _, v := range c.Variables {
		if v.Key == key {
			return nil, v
		}
	}
	return fmt.Errorf("variable key not found"), Variable{}
}

func (c *ConfigBody) Equals(c2 ConfigBody) bool {
	return reflect.DeepEqual(*c, c2)
}

func (c *ConfigBody) FromJSON(js []byte) (err error, rt ConfigBody) {
	var clss ConfigBody
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}
