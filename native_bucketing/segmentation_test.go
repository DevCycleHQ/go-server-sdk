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
		Audiences: []string{"test"},
	}
	_ = AudienceMatchFilter{
		filter: filter{
			Type:       "audienceMatch",
			Comparator: "!=",
		},
		Audiences: []string{"test"},
	}
	var filters = []BaseFilter{audienceMatchEqual}

	_ = OperatorFilter{
		Operator: &AudienceOperator{
			Operator: "and",
			Filters:  filters,
		},
	}
}

func TestEvaluateOperator_UserSubFilterInvalid(t *testing.T) {
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
			Type:       "user",
			SubType:    "myNewFilter",
			Comparator: "=",
		},
		Values: []interface{}{},
	}

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{userAllFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}

func TestEvaluateOperator_UserNewComparator(t *testing.T) {
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
			Type:       "user",
			SubType:    "email",
			Comparator: "wowNewComparator",
		},
		Values: []interface{}{},
	}

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{userAllFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}

func TestEvaluateOperator_UserFiltersAnd(t *testing.T) {
	platformData := PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}
	brooks := DVCPopulatedUser{
		Country:      "Canada",
		Email:        "brooks@big.lunch",
		PlatformData: platformData,
		AppVersion:   "2.0.2",
	}

	countryFilter := UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Canada"},
	}
	countryFilter.Initialize()
	emailFilter := UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "email",
			Comparator: "=",
		},
		Values: []interface{}{"dexter@smells.nice", "brooks@big.lunch"},
	}
	emailFilter.Initialize()
	appVerFilter := UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "appVersion",
			Comparator: ">",
		},
		Values: []interface{}{"1.0.0"},
	}
	appVerFilter.Initialize()

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{countryFilter, emailFilter, appVerFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
}

func TestEvaluateOperator_UserFiltersOr(t *testing.T) {
	platformData := PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}
	brooks := DVCPopulatedUser{
		Country:      "Canada",
		Email:        "brooks@big.lunch",
		PlatformData: platformData,
		AppVersion:   "2.0.2",
	}

	countryFilter := UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Canada"},
	}
	countryFilter.Initialize()
	emailFilter := UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "email",
			Comparator: "=",
		},
		Values: []interface{}{"dexter@smells.nice", "brooks@big.lunch"},
	}
	emailFilter.Initialize()
	appVerFilter := UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "appVersion",
			Comparator: ">",
		},
		Values: []interface{}{"1.0.0"},
	}
	appVerFilter.Initialize()

	result := _evaluateOperator(AudienceOperator{Operator: "or", Filters: MixedFilters{countryFilter, emailFilter, appVerFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
}

func TestEvaluateOperator_NestedAnd(t *testing.T) {
	// Nested filters currently do not have a model
	t.Fail()
}

func TestEvaluateOperator_NestedOr(t *testing.T) {
	// Nested filters currently do not have a model
	t.Fail()
}

func TestEvaluateOperator_AndCustomData(t *testing.T) {
	platformData := PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}
	brooks := DVCPopulatedUser{
		Country:      "Canada",
		Email:        "brooks@big.lunch",
		PlatformData: platformData,
		AppVersion:   "2.0.2",
	}

	countryFilter := UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Canada"},
	}
	countryFilter.Initialize()
	customDataFilter := CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: "!=",
			},
			Values: []interface{}{"Canada"},
		},
		DataKeyType: "String",
		DataKey:     "something",
	}
	customDataFilter.Initialize()

	result := _evaluateOperator(AudienceOperator{Operator: "or", Filters: MixedFilters{customDataFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
}

func TestEvaluateOperator_AndCustomDataMultiValue(t *testing.T) {
	platformData := PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}
	brooks := DVCPopulatedUser{
		Country:      "Canada",
		Email:        "brooks@big.lunch",
		PlatformData: platformData,
		AppVersion:   "2.0.2",
	}

	countryFilter := UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Canada"},
	}
	countryFilter.Initialize()
	customDataFilter := CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: "!=",
			},
			Values: []interface{}{"dataValue", "dataValue2"},
		},
		DataKeyType: "String",
		DataKey:     "something",
	}
	customDataFilter.Initialize()

	result := _evaluateOperator(AudienceOperator{Operator: "or", Filters: MixedFilters{customDataFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
}

func TestEvaluateOperator_AndPrivateCustomDataMultiValue(t *testing.T) {
	platformData := PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}
	brooks := DVCPopulatedUser{
		Country:      "Canada",
		Email:        "brooks@big.lunch",
		PlatformData: platformData,
		AppVersion:   "2.0.2",
		PrivateCustomData: map[string]interface{}{
			"testKey": "dataValue",
		},
	}

	countryFilter := UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Canada"},
	}
	countryFilter.Initialize()
	customDataFilter := CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: "!=",
			},
			Values: []interface{}{"dataValue", "dataValue2"},
		},
		DataKeyType: "String",
		DataKey:     "something",
	}
	customDataFilter.Initialize()

	result := _evaluateOperator(AudienceOperator{Operator: "or", Filters: MixedFilters{customDataFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}
