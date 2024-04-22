package variable_utils

import (
	"errors"
	"fmt"
	"reflect"
)

var ErrInvalidDefaultValue = errors.New("the default value for variable is not of type Boolean, Number, String, or JSON")

func ConvertDefaultValueType(value interface{}) interface{} {
	switch value := value.(type) {
	case int:
		return float64(value)
	case int8:
		return float64(value)
	case int16:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case uint:
		return float64(value)
	case uint8:
		return float64(value)
	case uint16:
		return float64(value)
	case uint32:
		return float64(value)
	case uint64:
		return float64(value)
	case float32:
		return float64(value)
	default:
		return value
	}
}

func VariableTypeFromValue(key string, value interface{}, allowNil bool) (varType string, err error) {
	switch value.(type) {
	case float64:
		return "Number", nil
	case string:
		return "String", nil
	case bool:
		return "Boolean", nil
	case map[string]any:
		return "JSON", nil
	case nil:
		if allowNil {
			return "", nil
		}
	}

	return "", fmt.Errorf("%w: %s", ErrInvalidDefaultValue, key)
}

func CompareTypes(value1 interface{}, value2 interface{}) bool {
	return reflect.TypeOf(value1) == reflect.TypeOf(value2)
}
