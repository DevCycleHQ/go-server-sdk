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
			Ftype:       "user",
			FsubType:    "email",
			Fcomparator: "=",
			values:      []interface{}{"brooks@big.lunch"},
		},
	}

	allFilter := FilterOrOperator{
		FilterClass: userEmailFilter,
	}

	result := _evaluateOperator(AudienceOperator{operator: "xylophone", filters: []FilterOrOperator{allFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}

func TestEvaluateOperator_AudienceFilterMatch(t *testing.T) {
	// This test is a bit tricky - need to figure out the inheritance setup.
	//userFilters := []UserFilter{
	//	{
	//		filter: filter{
	//			Ftype:       "user",
	//			FsubType:    "email",
	//			Fcomparator: "=",
	//			values:      []interface{}{"dexter@smells.nice", "brooks@big.lunch"}},
	//	},
	//	{
	//		filter: filter{
	//			Ftype:       "user",
	//			FsubType:    "country",
	//			Fcomparator: "=",
	//			values:      []interface{}{"Canada"}},
	//	},
	//	{
	//		filter: filter{
	//			Ftype:       "user",
	//			FsubType:    "appVersion",
	//			Fcomparator: ">",
	//			values:      []interface{}{"1.0.0"}},
	//	},
	//}
	//operator := AudienceOperator{
	//	operator: "and",
	//	filters:  userFilters,
	//}
	//audienceMatchEqual := AudienceMatchFilter{
	//	filter: filter{
	//		Ftype:       "audienceMatch",
	//		Fcomparator: "=",
	//	},
	//	audiences: []interface{}{"test"},
	//}
	//audienceMatchNotEqual := AudienceMatchFilter{
	//	filter: filter{
	//		Ftype:       "audienceMatch",
	//		Fcomparator: "!=",
	//	},
	//	audiences: []interface{}{"test"},
	//}
	//_filter := FilterOrOperator{FilterClass: audienceMatchEqual}
	//var filters = []FilterOrOperator{_filter}
	//op := FilterOrOperator{
	//	OperatorClass: AudienceOperator{
	//		operator: "and",
	//		filters:  filters,
	//	},
	//}

}
