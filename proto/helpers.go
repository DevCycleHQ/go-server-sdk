package proto

import "encoding/json"

func (variable SDKVariable_PB) GetValue() interface{} {
	switch variable.Type {
	case VariableType_PB_Boolean:
		return variable.BoolValue
	case VariableType_PB_Number:
		return variable.DoubleValue
	case VariableType_PB_String:
		return variable.StringValue
	case VariableType_PB_JSON:
		var result interface{}
		json.Unmarshal([]byte(variable.StringValue), &result)
		return result
	}
	return nil
}
