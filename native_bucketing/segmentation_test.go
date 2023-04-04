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
	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: []BaseFilter{}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
	result = _evaluateOperator(AudienceOperator{Operator: "or", Filters: []BaseFilter{}}, nil, brooks, nil)
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
			Type:       "all",
			Comparator: "=",
		},
		Values: []interface{}{},
	}

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: []BaseFilter{userAllFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
	result = _evaluateOperator(AudienceOperator{Operator: "or", Filters: []BaseFilter{userAllFilter}}, nil, brooks, nil)
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
			Type:       "myNewFilter",
			Comparator: "=",
		},
		Values: []interface{}{},
	}

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: []BaseFilter{userAllFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
	result = _evaluateOperator(AudienceOperator{Operator: "or", Filters: []BaseFilter{userAllFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}

func TestEvaluateOperator_InvalidComparator(t *testing.T) {
	platformData := PlatformData{
		Platform:        "iOS",
		PlatformVersion: "10.3.1",
	}
	brooks := DVCPopulatedUser{
		Country:      "Canada",
		Email:        "brooks@big.lunch",
		PlatformData: platformData,
	}
	userEmailFilter := UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "email",
			Comparator: "=",
		},
		Values: []interface{}{"brooks@big.lunch"},
	}

	result := _evaluateOperator(AudienceOperator{Operator: "xylophone", Filters: []BaseFilter{userEmailFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}

func TestEvaluateOperator_AudienceFilterMatch(t *testing.T) {
	// This test is a bit tricky - need to figure out the inheritance setup.
	userFilters := MixedFilters{
		UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "email",
				Comparator: "=",
			},
			Values: []interface{}{"dexter@smells.nice", "brooks@big.lunch"},
		},
		UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "country",
				Comparator: "=",
			},
			Values: []interface{}{"Canada"}},
		UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "appVersion",
				Comparator: ">",
			},
			Values: []interface{}{"1.0.0"}},
	}
	_ = AudienceOperator{
		Operator: "and",
		Filters:  userFilters,
	}
	audienceMatchEqual := AudienceMatchFilter{
		filter: filter{
			Type:       "audienceMatch",
			Comparator: "=",
		},
		audiences: []interface{}{"test"},
	}
	_ = AudienceMatchFilter{
		filter: filter{
			Type:       "audienceMatch",
			Comparator: "!=",
		},
		audiences: []interface{}{"test"},
	}
	var filters = []BaseFilter{audienceMatchEqual}

	_ = OperatorFilter{
		Operator: &AudienceOperator{
			Operator: "and",
			Filters:  filters,
		},
	}

}
