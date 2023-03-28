package native_bucketing

import (
	"encoding/json"
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
	Project        PublicProject     `json:"project"`
	Environment    PublicEnvironment `json:"environment"`
	Features       []Feature         `json:"features"`
	Variables      []Variable        `json:"variables"`
	VariableHashes map[string]int64  `json:"variableHashes"`
}

func (c *ConfigBody) Equals(c2 ConfigBody) bool {
	return reflect.DeepEqual(*c, c2)
}

func (c *ConfigBody) FromJSON(js string) (err error, rt ConfigBody) {
	var clss ConfigBody
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}
