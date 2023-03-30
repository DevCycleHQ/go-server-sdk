package native_bucketing

import "testing"

func TestSegmentation_EvaluateOperator_FailEmpty(t *testing.T) {

	platformData := PlatformData{
		Platform:        "iOS",
		PlatformVersion: "10.3.1",
	}
	brooks := DVCPopulatedUser{
		Country:      "Canada",
		Email:        "brooks@big.lunch",
		PlatformData: platformData,
	}
	result := _evaluateOperator(AudienceOperator{operator: "and", filters: []FilterOrOperator{}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
	result = _evaluateOperator(AudienceOperator{operator: "or", filters: []FilterOrOperator{}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}

func TestSegementation_EvaluateOperator_PassAll(t *testing.T) {
	platformData := PlatformData{
		Platform:        "iOS",
		PlatformVersion: "10.3.1",
	}
	brooks := DVCPopulatedUser{
		Country:      "Canada",
		Email:        "brooks@big.lunch",
		PlatformData: platformData,
	}

	userAllFilter := UserFilter{
		filter: filter{
			Ftype:       "all",
			Fcomparator: "=",
			values:      []interface{}{},
		},
	}

	allFilter := FilterOrOperator{
		FilterClass: userAllFilter,
	}
	result := _evaluateOperator(AudienceOperator{operator: "and", filters: []FilterOrOperator{allFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
	result = _evaluateOperator(AudienceOperator{operator: "or", filters: []FilterOrOperator{allFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
}

func TestSegementation_EvaluateOperator_UnknownFilter(t *testing.T) {
	platformData := PlatformData{
		Platform:        "iOS",
		PlatformVersion: "10.3.1",
	}
	brooks := DVCPopulatedUser{
		Country:      "Canada",
		Email:        "brooks@big.lunch",
		PlatformData: platformData,
	}

	userAllFilter := UserFilter{
		filter: filter{
			Ftype:       "myNewFilter",
			Fcomparator: "=",
			values:      []interface{}{},
		},
	}

	allFilter := FilterOrOperator{
		FilterClass: userAllFilter,
	}
	result := _evaluateOperator(AudienceOperator{operator: "and", filters: []FilterOrOperator{allFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
	result = _evaluateOperator(AudienceOperator{operator: "or", filters: []FilterOrOperator{allFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}
