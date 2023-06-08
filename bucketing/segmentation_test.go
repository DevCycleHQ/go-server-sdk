package bucketing

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/devcyclehq/go-server-sdk/v2/api"
)

var brooks = api.PopulatedUser{
	User: api.User{
		Country:    "Canada",
		Email:      "brooks@big.lunch",
		AppVersion: "2.0.2",
	},
	PlatformData: &api.PlatformData{
		Platform:        "iOS",
		PlatformVersion: "10.3.1",
	},
}

func _evaluateOperator(operator FilterOrOperator, audiences map[string]NoIdAudience, user api.PopulatedUser, clientCustomData map[string]interface{}) bool {
	return operator.Evaluate(audiences, user, clientCustomData)
}

func TestSegmentation_EvaluateOperator_FailEmpty(t *testing.T) {
	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: []FilterOrOperator{}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
	result = _evaluateOperator(AudienceOperator{Operator: "or", Filters: []FilterOrOperator{}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}

func TestSegmentation_EvaluateOperator_PassAll(t *testing.T) {
	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: []FilterOrOperator{AllFilter{}}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
	result = _evaluateOperator(AudienceOperator{Operator: "or", Filters: []FilterOrOperator{AllFilter{}}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
}

func TestSegmentation_EvaluateOperator_UnknownFilter(t *testing.T) {
	userAllFilter := &UserFilter{
		filter: filter{
			Type:       "myNewFilter",
			Comparator: "=",
		},
		Values: []interface{}{},
	}

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: []FilterOrOperator{userAllFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
	result = _evaluateOperator(AudienceOperator{Operator: "or", Filters: []FilterOrOperator{userAllFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}

func TestEvaluateOperator_InvalidComparator(t *testing.T) {
	userEmailFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "email",
			Comparator: "=",
		},
		Values: []interface{}{"brooks@big.lunch"},
	}

	result := _evaluateOperator(AudienceOperator{Operator: "xylophone", Filters: []FilterOrOperator{userEmailFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}

func TestEvaluateOperator_AudienceFilterMatch(t *testing.T) {
	countryFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
			Operator:   OperatorAnd,
		},
		Values: []interface{}{"Canada"},
	}
	require.NoError(t, countryFilter.Initialize())
	emailFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "email",
			Comparator: "=",
			Operator:   OperatorAnd,
		},
		Values: []interface{}{
			"dexter@smells.nice",
			"brooks@big.lunch",
		},
	}
	require.NoError(t, emailFilter.Initialize())
	versionFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "appVersion",
			Comparator: ">",
			Operator:   OperatorAnd,
		},
		Values: []interface{}{
			"1.0.0",
		},
	}
	require.NoError(t, versionFilter.Initialize())

	audience := Audience{
		NoIdAudience: NoIdAudience{
			Filters: &AudienceOperator{
				Operator: OperatorAnd,
				Filters: []FilterOrOperator{
					countryFilter,
					emailFilter,
					versionFilter,
				},
			},
		},
		Id: "test",
	}

	testCases := []struct {
		name      string
		filters   []FilterOrOperator
		audiences map[string]NoIdAudience
		expected  bool
	}{
		{
			name: "audienceMatchFilter - in the audience",
			filters: []FilterOrOperator{&AudienceMatchFilter{
				filter: filter{
					Type:       "audienceMatch",
					Comparator: "=",
					Operator:   OperatorAnd,
				},
				Audiences: []string{"test"},
			}},
			audiences: map[string]NoIdAudience{"test": audience.NoIdAudience},
			expected:  true,
		},
		{
			name: "audienceMatchFilter - not in the audience",
			filters: []FilterOrOperator{&AudienceMatchFilter{
				filter: filter{
					Type:       "audienceMatch",
					Comparator: "!=",
					Operator:   OperatorAnd,
				},
				Audiences: []string{"test"},
			}},
			audiences: map[string]NoIdAudience{"test": audience.NoIdAudience},
			expected:  false,
		},
		{
			name: "audienceMatchFilter - audience ID not in list",
			filters: []FilterOrOperator{&AudienceMatchFilter{
				filter: filter{
					Type:       "audienceMatch",
					Comparator: "==",
					Operator:   OperatorAnd,
				},
				Audiences: []string{"test"},
			}},
			audiences: map[string]NoIdAudience{},
			expected:  false,
		},
		{
			name: "audienceMatchFilter - audience ID not in list",
			filters: []FilterOrOperator{&AudienceMatchFilter{
				filter: filter{
					Type:       "audienceMatch",
					Comparator: "==",
					Operator:   OperatorAnd,
				},
				Audiences: []string{"someOtherAudienceID"},
			}},
			audiences: map[string]NoIdAudience{"test": audience.NoIdAudience},
			expected:  false,
		},
	}

	for _, tc := range testCases {
		result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: tc.filters}, tc.audiences, brooks, nil)
		if result != tc.expected {
			t.Errorf("%v - Expected %t, got %t", tc.name, tc.expected, result)
		}
	}
}

func TestEvaluateOperator_AudienceFilterForPlatforms(t *testing.T) {
	testCases := []struct {
		name     string
		filter   *UserFilter
		expected bool
	}{
		{
			name: "should filter all Android TV audiences properly if it is included in data - 3 elements",
			filter: &UserFilter{
				filter: filter{
					Type:       "user",
					SubType:    "platform",
					Comparator: "=",
					Operator:   OperatorAnd,
				},
				Values: []interface{}{"Android", "Fire TV", "Android TV"},
			},
			expected: true,
		},
		{
			name: "should filter all Android TV audiences properly if it is included in data - 1 elements",
			filter: &UserFilter{

				filter: filter{
					Type:       "user",
					SubType:    "platform",
					Comparator: "=",
					Operator:   OperatorAnd,
				},
				Values: []interface{}{"Fire TV", "Android TV"},
			},
			expected: true,
		},
		{
			name: "should filter all Android TV audiences properly if it is included in data - 1 element",
			filter: &UserFilter{
				filter: filter{
					Type:       "user",
					SubType:    "platform",
					Comparator: "=",
					Operator:   OperatorAnd,
				},
				Values: []interface{}{"Android TV"},
			},
			expected: true,
		},
		{
			name: "should filter all Android TV audiences properly if it is included in data - similar but !=",
			filter: &UserFilter{
				filter: filter{
					Type:       "user",
					SubType:    "platform",
					Comparator: "=",
					Operator:   OperatorAnd,
				},
				Values: []interface{}{"Android"},
			},
			expected: false,
		},
		{
			name: "should filter all Android TV audiences properly if it is included in data - different platform",
			filter: &UserFilter{
				filter: filter{
					Type:       "user",
					SubType:    "platform",
					Comparator: "=",
					Operator:   OperatorAnd,
				},
				Values: []interface{}{"iOS"},
			},
			expected: false,
		},
	}

	testUser := api.PopulatedUser{
		User: api.User{
			UserId: "9999",
		},
		PlatformData: &api.PlatformData{
			Platform: "Android TV",
		},
	}

	for _, tc := range testCases {
		require.NoError(t, tc.filter.Initialize())
		audienceID := "test"
		// create an audience with the test filter
		audience := Audience{
			NoIdAudience: NoIdAudience{
				Filters: &AudienceOperator{
					Operator: OperatorAnd,
					Filters: []FilterOrOperator{
						tc.filter,
					},
				},
			},
			Id: audienceID,
		}

		operatorFilters := []FilterOrOperator{&AudienceMatchFilter{
			filter: filter{
				Type:       "audienceMatch",
				Comparator: ComparatorEqual,
				Operator:   OperatorAnd,
			},
			Audiences: []string{audienceID},
		}}

		// Make sure that we are matching to the audience properly based on platform
		result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: operatorFilters}, map[string]NoIdAudience{audienceID: audience.NoIdAudience}, testUser, nil)
		if result != tc.expected {
			t.Errorf("%v - Expected %t, got %t", tc.name, tc.expected, result)
		}
	}
}

func TestEvaluateOperator_AudienceFilterMatchMultipleValuesAND(t *testing.T) {
	countryCANFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
			Operator:   OperatorAnd,
		},
		Values: []interface{}{"Canada"},
	}
	require.NoError(t, countryCANFilter.Initialize())

	countryUSAFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
			Operator:   OperatorAnd,
		},
		Values: []interface{}{"USA"},
	}
	require.NoError(t, countryUSAFilter.Initialize())

	emailFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "email",
			Comparator: "=",
			Operator:   OperatorAnd,
		},
		Values: []interface{}{
			"dexter@smells.nice",
			"brooks@big.lunch",
		},
	}
	require.NoError(t, emailFilter.Initialize())

	versionFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "appVersion",
			Comparator: ">",
			Operator:   OperatorAnd,
		},
		Values: []interface{}{
			"1.0.0",
		},
	}
	require.NoError(t, versionFilter.Initialize())

	audience := Audience{
		NoIdAudience: NoIdAudience{
			Filters: &AudienceOperator{
				Operator: OperatorAnd,
				Filters: []FilterOrOperator{
					countryCANFilter,
					emailFilter,
					versionFilter,
				},
			},
		},
		Id: "test",
	}

	audience2 := Audience{
		NoIdAudience: NoIdAudience{
			Filters: &AudienceOperator{
				Operator: OperatorAnd,
				Filters: []FilterOrOperator{
					countryUSAFilter,
				},
			},
		},
		Id: "test2",
	}

	// audience match filters only support = and != comparators
	// a user is either in one of the audiences for `=` or they have to not be in any of the audiences for `!=`
	testCases := []struct {
		name      string
		filters   []FilterOrOperator
		audiences map[string]NoIdAudience
		expected  bool
	}{
		{
			name: "should pass seg for an AND operator with multiple values",
			filters: []FilterOrOperator{&AudienceMatchFilter{
				filter: filter{
					Type:       "audienceMatch",
					Comparator: ComparatorEqual,
					Operator:   OperatorAnd,
				},
				Audiences: []string{"test", "test2"},
			}},
			audiences: map[string]NoIdAudience{"test": audience.NoIdAudience, "test2": audience2.NoIdAudience},
			expected:  true,
		},
		{
			name: "audienceMatchFilter - should not pass seg for an AND operator with multiple values",
			filters: []FilterOrOperator{&AudienceMatchFilter{
				filter: filter{
					Type:       "audienceMatch",
					Comparator: ComparatorNotEqual,
					Operator:   OperatorAnd,
				},
				Audiences: []string{"test", "test2"},
			}},
			audiences: map[string]NoIdAudience{"test": audience.NoIdAudience, "test2": audience2.NoIdAudience},
			expected:  false,
		},
	}

	for _, tc := range testCases {
		result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: tc.filters}, tc.audiences, brooks, nil)
		if result != tc.expected {
			t.Errorf("%v - Expected %t, got %t", tc.name, tc.expected, result)
		}
	}
}

func TestEvaluateOperator_AudienceNested(t *testing.T) {
	countryFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Canada"},
	}
	require.NoError(t, countryFilter.Initialize())
	audienceInner := NoIdAudience{
		Filters: &AudienceOperator{
			Operator: OperatorAnd,
			Filters:  []FilterOrOperator{countryFilter},
		},
	}
	audienceOuter := NoIdAudience{
		Filters: &AudienceOperator{
			Operator: OperatorAnd,
			Filters: []FilterOrOperator{&AudienceMatchFilter{
				filter: filter{
					Type:       "audienceMatch",
					Comparator: "=",
					Operator:   OperatorAnd,
				},
				Audiences: []string{"inner"},
			}},
		},
	}
	audiences := map[string]NoIdAudience{
		"outer": audienceOuter,
		"inner": audienceInner,
	}
	operator := &AudienceOperator{
		Operator: OperatorAnd,
		Filters: []FilterOrOperator{&AudienceMatchFilter{
			filter: filter{
				Type:       "audienceMatch",
				Comparator: "=",
				Operator:   OperatorAnd,
			},
			Audiences: []string{"outer"},
		}},
	}
	result := _evaluateOperator(operator, audiences, brooks, nil)
	require.True(t, result)

	operator2 := &AudienceOperator{
		Operator: OperatorAnd,
		Filters: []FilterOrOperator{
			countryFilter,
			&AudienceMatchFilter{
				filter: filter{
					Type:       "audienceMatch",
					Comparator: "=",
					Operator:   OperatorAnd,
				},
				Audiences: []string{"inner"},
			},
		},
	}
	result = _evaluateOperator(operator2, audiences, brooks, nil)
	require.True(t, result)
}

func TestEvaluateOperator_UserSubFilterInvalid(t *testing.T) {
	userAllFilter := &UserFilter{
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
	userAllFilter := &UserFilter{
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
	countryFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Canada"},
	}
	require.NoError(t, countryFilter.Initialize())
	emailFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "email",
			Comparator: "=",
		},
		Values: []interface{}{"dexter@smells.nice", "brooks@big.lunch"},
	}
	require.NoError(t, emailFilter.Initialize())
	appVerFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "appVersion",
			Comparator: ">",
		},
		Values: []interface{}{"1.0.0"},
	}
	require.NoError(t, appVerFilter.Initialize())

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{countryFilter, emailFilter, appVerFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
}

func TestEvaluateOperator_UserFiltersOr(t *testing.T) {
	countryFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Banada"},
	}
	require.NoError(t, countryFilter.Initialize())
	emailFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "email",
			Comparator: "=",
		},
		Values: []interface{}{"dexter@smells.nice"},
	}
	require.NoError(t, emailFilter.Initialize())
	appVerFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "appVersion",
			Comparator: ">",
		},
		Values: []interface{}{"1.0.0"},
	}
	require.NoError(t, appVerFilter.Initialize())

	result := _evaluateOperator(AudienceOperator{Operator: "or", Filters: MixedFilters{countryFilter, emailFilter, appVerFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
}

func TestEvaluateOperator_NestedAnd(t *testing.T) {
	countryFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Canada"},
	}
	require.NoError(t, countryFilter.Initialize())
	emailFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "email",
			Comparator: "=",
		},
		Values: []interface{}{"dexter@smells.nice", "brooks@big.lunch"},
	}
	require.NoError(t, emailFilter.Initialize())
	appVerFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "appVersion",
			Comparator: ">",
		},
		Values: []interface{}{"1.0.0"},
	}
	require.NoError(t, appVerFilter.Initialize())

	nestedOperator := &AudienceOperator{
		Operator: "and",
		Filters:  MixedFilters{countryFilter, emailFilter, appVerFilter},
	}
	topLevelFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "!=",
		},
		Values: []interface{}{"Nanada"},
	}
	require.NoError(t, topLevelFilter.Initialize())

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{nestedOperator, topLevelFilter}}, nil, brooks, nil)
	require.True(t, result)

	// If the second AND filter fails, should fail to match
	topLevelFilter.filter.Comparator = "="
	result = _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{nestedOperator, topLevelFilter}}, nil, brooks, nil)
	require.False(t, result)
}

func TestEvaluateOperator_NestedOr(t *testing.T) {
	countryFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Nanada"},
	}
	require.NoError(t, countryFilter.Initialize())
	emailFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "email",
			Comparator: "=",
		},
		Values: []interface{}{"dexter@smells.nice", "brooks@big.lunch"},
	}
	require.NoError(t, emailFilter.Initialize())
	appVerFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "appVersion",
			Comparator: "=",
		},
		Values: []interface{}{"1.0.0"},
	}
	require.NoError(t, appVerFilter.Initialize())

	nestedOperator := &AudienceOperator{
		Operator: "or",
		Filters:  MixedFilters{countryFilter, emailFilter, appVerFilter},
	}
	topLevelFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Nanada"},
	}

	result := _evaluateOperator(AudienceOperator{Operator: "or", Filters: MixedFilters{nestedOperator, topLevelFilter}}, nil, brooks, nil)
	require.True(t, result)
}

func TestEvaluateOperator_AndCustomData(t *testing.T) {
	countryFilter := &UserFilter{
		filter: filter{
			Type:       "user",
			SubType:    "country",
			Comparator: "=",
		},
		Values: []interface{}{"Canada"},
	}
	require.NoError(t, countryFilter.Initialize())
	customDataFilter := &CustomDataFilter{
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
	require.NoError(t, customDataFilter.Initialize())

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{countryFilter, customDataFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
}

func TestEvaluateOperator_AndCustomDataPrioritizeUserData(t *testing.T) {
	platformData := &api.PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}
	testUser := api.PopulatedUser{
		User: api.User{
			Country:    "Canada",
			Email:      "brooks@big.lunch",
			AppVersion: "2.0.2",
			CustomData: map[string]interface{}{
				"user_tier": "Gold",
			},
		},
		PlatformData: platformData,
	}
	customDataFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorEqual,
			},
			Values: []interface{}{"Gold"},
		},
		DataKeyType: "String",
		DataKey:     "user_tier",
	}
	require.NoError(t, customDataFilter.Initialize())

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{customDataFilter}}, nil, testUser, map[string]interface{}{"user_tier": "Silver"})
	if !result {
		t.Error("Expected true, got false")
	}

	testUser = api.PopulatedUser{
		User: api.User{
			Country:    "Canada",
			Email:      "brooks@big.lunch",
			AppVersion: "2.0.2",
			CustomData: map[string]interface{}{
				"user_tier": "Silver",
			},
		},
		PlatformData: platformData,
	}

	// the user has a custom data value of "Silver" but the filter is looking for "Gold", the clientCustomData is irrelevant
	result = _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{customDataFilter}}, nil, testUser, map[string]interface{}{"user_tier": "Gold"})
	if result {
		t.Error("Expected false, got true")
	}
}

func TestEvaluateOperator_AndCustomDataMultiValue(t *testing.T) {
	platformData := &api.PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}
	brooks := api.PopulatedUser{
		User: api.User{
			Country:    "Canada",
			Email:      "brooks@big.lunch",
			AppVersion: "2.0.2",
			CustomData: map[string]interface{}{
				"something": "dataValue",
			},
		},
		PlatformData: platformData,
	}
	customDataFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: "=",
			},
			Values: []interface{}{"dataValue", "dataValue2"},
		},
		DataKeyType: "String",
		DataKey:     "something",
	}
	require.NoError(t, customDataFilter.Initialize())

	result := _evaluateOperator(AudienceOperator{Operator: "or", Filters: MixedFilters{customDataFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}

	result = _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{customDataFilter}}, nil, brooks, nil)
	if !result {
		t.Error("Expected true, got false")
	}
}

func TestEvaluateOperator_AndPrivateCustomDataMultiValue(t *testing.T) {
	var brooks = api.PopulatedUser{
		User: api.User{
			Country:    "Canada",
			Email:      "brooks@big.lunch",
			AppVersion: "2.0.2",
			PrivateCustomData: map[string]interface{}{
				"testKey": "dataValue",
			},
		},
		PlatformData: &api.PlatformData{
			Platform:        "iOS",
			PlatformVersion: "2.0.0",
		},
	}

	customDataFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: "!=",
			},
			Values: []interface{}{"dataValue", "dataValue2"},
		},
		DataKeyType: "String",
		DataKey:     "testKey",
	}
	require.NoError(t, customDataFilter.Initialize())

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{customDataFilter}}, nil, brooks, nil)
	if result {
		t.Error("Expected false, got true")
	}
}

func TestEvaluateOperator_AndClientCustomDataMultiValue(t *testing.T) {
	platformData := &api.PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}
	brooks := api.PopulatedUser{
		User: api.User{
			Country:    "Canada",
			Email:      "brooks@big.lunch",
			AppVersion: "2.0.2",
		},
		PlatformData: platformData,
	}
	customDataFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: "=",
			},
			Values: []interface{}{"dataValue", "dataValue2"},
		},
		DataKeyType: "String",
		DataKey:     "something",
	}
	require.NoError(t, customDataFilter.Initialize())

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{customDataFilter}}, nil, brooks, map[string]interface{}{"something": "dataValue"})
	if !result {
		t.Error("Expected true, got false")
	}

	// test clientCustomData with different key
	result = _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{customDataFilter}}, nil, brooks, map[string]interface{}{"otherKey": "dataValue"})
	if result {
		t.Error("Expected true, got false")
	}

	// test empty clientCustomData map
	result = _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{customDataFilter}}, nil, brooks, map[string]interface{}{})
	if result {
		t.Error("Expected true, got false")
	}
}

func TestEvaluateOperator_MultiEqualCustomDataFilters(t *testing.T) {
	strFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorEqual,
			},
			Values: []interface{}{"value"},
		},
		DataKey:     "strKey",
		DataKeyType: "String",
	}
	require.NoError(t, strFilter.Initialize())
	require.NoError(t, strFilter.UserFilter.Initialize())

	numFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorEqual,
			},
			Values: []interface{}{float64(0)},
		},
		DataKey:     "numKey",
		DataKeyType: "Number",
	}
	require.NoError(t, numFilter.Initialize())
	require.NoError(t, numFilter.UserFilter.Initialize())

	boolFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorEqual,
			},
			Values: []interface{}{false},
		},
		DataKey:     "boolKey",
		DataKeyType: "Boolean",
	}
	require.NoError(t, boolFilter.Initialize())
	require.NoError(t, boolFilter.UserFilter.Initialize())

	platformData := &api.PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}

	userAllValues := api.PopulatedUser{
		User: api.User{
			UserId: "12345",
			CustomData: map[string]interface{}{
				"strKey": "value", "boolKey": false, "numKey": float64(0),
			},
		},
		PlatformData: platformData,
	}
	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{strFilter, numFilter, boolFilter}}, nil, userAllValues, nil)
	if !result {
		t.Errorf("Test 'should return true if all filters are equal' failed. Expected %t, got %t", false, result)
	}

	userMissingKey := api.PopulatedUser{
		User: api.User{
			UserId: "12345",
			CustomData: map[string]interface{}{
				"strKey": "value", "boolKey": false,
			},
		},
		PlatformData: platformData,
	}

	result = _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{strFilter, numFilter, boolFilter}}, nil, userMissingKey, nil)
	if result {
		t.Errorf("Test 'should return false if one custom data key is missing' failed. Expected %t, got %t", false, result)
	}

	userNoCustomData := api.PopulatedUser{
		User: api.User{
			UserId:     "12345",
			CustomData: map[string]interface{}{},
		},
		PlatformData: platformData,
	}

	result = _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{strFilter, numFilter, boolFilter}}, nil, userNoCustomData, map[string]interface{}{"strKey": "value", "boolKey": false, "numKey": float64(0)})
	if !result {
		t.Errorf("Test 'should return true if all data keys are in clientCustomData' failed. Expected %t, got %t", false, result)
	}
}

func TestEvaluateOperator_MultiNotEqualCustomDataFilters(t *testing.T) {
	strFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorEqual,
			},
			Values: []interface{}{"value"},
		},
		DataKey:     "strKey",
		DataKeyType: "String",
	}
	require.NoError(t, strFilter.Initialize())
	require.NoError(t, strFilter.UserFilter.Initialize())

	numFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorNotEqual,
			},
			Values: []interface{}{float64(0)},
		},
		DataKey:     "numKey",
		DataKeyType: "Number",
	}
	require.NoError(t, numFilter.Initialize())
	require.NoError(t, numFilter.UserFilter.Initialize())

	boolFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorEqual,
			},
			Values: []interface{}{false},
		},
		DataKey:     "boolKey",
		DataKeyType: "Boolean",
	}
	require.NoError(t, boolFilter.Initialize())
	require.NoError(t, boolFilter.UserFilter.Initialize())

	platformData := &api.PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}

	userAllValues := api.PopulatedUser{
		User: api.User{
			UserId: "12345",
			CustomData: map[string]interface{}{
				"strKey": "value", "boolKey": false, "numKey": float64(0),
			},
		},
		PlatformData: platformData,
	}

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{strFilter, numFilter, boolFilter}}, nil, userAllValues, nil)
	if result {
		t.Errorf("Test 'should return false all the keys match with not equal filter value' failed. Expected %t, got %t", false, result)
	}

	userMissingKey := api.PopulatedUser{
		User: api.User{
			UserId: "12345",
			CustomData: map[string]interface{}{
				"strKey": "value", "boolKey": false,
			},
		},
		PlatformData: platformData,
	}

	result = _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{strFilter, numFilter, boolFilter}}, nil, userMissingKey, nil)
	if !result {
		t.Errorf("Test 'should return true if one custom data key is missing with not equal filter value' failed. Expected %t, got %t", true, result)
	}

	userNoCustomData := api.PopulatedUser{
		User: api.User{
			UserId:     "12345",
			CustomData: map[string]interface{}{},
		},
		PlatformData: platformData,
	}

	result = _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{strFilter, numFilter, boolFilter}}, nil, userNoCustomData, nil)
	if result {
		t.Errorf("Test 'should return false if no custom data is provided with just not equal filter value' failed. Expected %t, got %t", false, result)
	}
}

func TestEvaluateOperator_MultiNotExistsCustomDataFilters(t *testing.T) {
	strFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorEqual,
			},
			Values: []interface{}{"value"},
		},
		DataKey:     "strKey",
		DataKeyType: "String",
	}
	require.NoError(t, strFilter.Initialize())
	require.NoError(t, strFilter.UserFilter.Initialize())

	numFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorNotExist,
			},
			Values: []interface{}{float64(0)},
		},
		DataKey:     "numKey",
		DataKeyType: "Number",
	}
	require.NoError(t, numFilter.Initialize())
	require.NoError(t, numFilter.UserFilter.Initialize())

	boolFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorEqual,
			},
			Values: []interface{}{false},
		},
		DataKey:     "boolKey",
		DataKeyType: "Boolean",
	}
	require.NoError(t, boolFilter.Initialize())
	require.NoError(t, boolFilter.UserFilter.Initialize())

	platformData := &api.PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}

	userAllValues := api.PopulatedUser{
		User: api.User{
			UserId: "12345",
			CustomData: map[string]interface{}{
				"strKey": "value", "boolKey": false, "numKey": float64(0),
			},
		},
		PlatformData: platformData,
	}

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{strFilter, numFilter, boolFilter}}, nil, userAllValues, nil)
	if result {
		t.Errorf("Test 'should return false all the keys match with !exists filter value' failed. Expected %t, got %t", false, result)
	}

	userMissingKey := api.PopulatedUser{
		User: api.User{
			UserId: "12345",
			CustomData: map[string]interface{}{
				"strKey": "value", "boolKey": false,
			},
		},
		PlatformData: platformData,
	}

	result = _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{strFilter, numFilter, boolFilter}}, nil, userMissingKey, nil)
	if !result {
		t.Errorf("Test 'should return true if one custom data key is missing with !exists filter value' failed. Expected %t, got %t", true, result)
	}

	userNoCustomData := api.PopulatedUser{
		User: api.User{
			UserId:     "12345",
			CustomData: map[string]interface{}{},
		},
		PlatformData: platformData,
	}

	result = _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{strFilter, numFilter, boolFilter}}, nil, userNoCustomData, nil)
	if result {
		t.Errorf("Test 'should return false if no custom data is provided with just !exists filter value' failed. Expected %t, got %t", false, result)
	}
}

func TestEvaluateOperator_OptIn(t *testing.T) {
	// The OptIn feature is not supported by Local Bucketing but we do need to make sure it return false and doesn't generate a panic
	optInFilter := &UserFilter{
		filter: filter{
			Type:       "optIn",
			Comparator: ComparatorEqual,
		},
		Values: []interface{}{},
	}
	require.NoError(t, optInFilter.Initialize())

	var optInUser = api.PopulatedUser{
		User: api.User{
			Country:    "Canada",
			Email:      "brooks@big.lunch",
			AppVersion: "2.0.2",
		},
		PlatformData: &api.PlatformData{
			Platform:        "iOS",
			PlatformVersion: "2.0.0",
		},
	}

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{optInFilter}}, nil, optInUser, nil)
	if result {
		t.Error("should fail optIn filter when feature in optIns and isOptInEnabled")
	}
}

func TestEvaluateOperator_MultiCustomDataFiltersMergedCustomData(t *testing.T) {
	strFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorEqual,
			},
			Values: []interface{}{"value"},
		},
		DataKey:     "strKey",
		DataKeyType: "String",
	}
	require.NoError(t, strFilter.Initialize())
	require.NoError(t, strFilter.UserFilter.Initialize())

	numFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorEqual,
			},
			Values: []interface{}{float64(0)},
		},
		DataKey:     "numKey",
		DataKeyType: "Number",
	}
	require.NoError(t, numFilter.Initialize())
	require.NoError(t, numFilter.UserFilter.Initialize())

	boolFilter := &CustomDataFilter{
		UserFilter: &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: ComparatorEqual,
			},
			Values: []interface{}{false},
		},
		DataKey:     "boolKey",
		DataKeyType: "Boolean",
	}
	require.NoError(t, boolFilter.Initialize())
	require.NoError(t, boolFilter.UserFilter.Initialize())

	platformData := &api.PlatformData{
		Platform:        "iOS",
		PlatformVersion: "2.0.0",
	}

	userAllValues := api.PopulatedUser{
		User: api.User{
			UserId: "12345",
			CustomData: map[string]interface{}{
				"strKey": "value",
			},
			PrivateCustomData: map[string]interface{}{"numKey": float64(0)},
		},
		PlatformData: platformData,
	}

	result := _evaluateOperator(AudienceOperator{Operator: "and", Filters: MixedFilters{strFilter, numFilter, boolFilter}}, nil, userAllValues, map[string]interface{}{"boolKey": false})
	if !result {
		t.Errorf("Test 'should return true all the custom data matches from customData, privateCustomData nd clientCustomData' failed. Expected %t, got %t", true, result)
	}
}

func Test_checkVersionValue(t *testing.T) {
	testCases := []struct {
		filterVersion string
		version       string
		operator      string
		expected      bool
	}{
		{
			filterVersion: "1.2.3",
			version:       "1.2.3",
			operator:      "==",
			expected:      true,
		},
		{
			filterVersion: "1.2.3",
			version:       "2.3.4",
			operator:      "==",
			expected:      false,
		},
		{
			filterVersion: "1.2.3",
			version:       "2.3.4",
			operator:      ">",
			expected:      true,
		},
		{
			filterVersion: "2.3.4",
			version:       "1.2.3",
			operator:      ">",
			expected:      false,
		},
		{
			filterVersion: "2.3.4",
			version:       "1.2.3",
			operator:      "<",
			expected:      true,
		},
		{
			filterVersion: "1.2.3",
			version:       "2.3.4",
			operator:      "<",
			expected:      false,
		},
	}

	for _, tc := range testCases {
		got := checkVersionValue(tc.filterVersion, tc.version, tc.operator)
		if got != tc.expected {
			t.Errorf("checkVersionValue(%s, %s, %s) = %v; want %v", tc.filterVersion, tc.version, tc.operator, got, tc.expected)
		}
	}
}

func Test_ConvertToSemanticVersion(t *testing.T) {
	testCases := []struct {
		input  string
		output string
	}{
		{
			input:  "1.2.3",
			output: "1.2.3",
		},
		{
			input:  "1.2",
			output: "1.2.0",
		},
		{
			input:  "1",
			output: "1.0.0",
		},
		{
			input:  "1..3",
			output: "1.0.3",
		},
		{
			input:  "1.2.3.4",
			output: "1.2.3.4",
		},
	}

	for _, tc := range testCases {
		got := convertToSemanticVersion(tc.input)
		if got != tc.output {
			t.Errorf("convertToSemanticVersion(%s) = %s; want %s", tc.input, got, tc.output)
		}
	}
}

func Test_CheckNumberFilter(t *testing.T) {
	type NumberTestCase struct {
		num        float64
		filterNums []float64
		operator   string
		want       bool
	}

	testCases := []NumberTestCase{
		{num: math.NaN(), filterNums: []float64{}, operator: "", want: false},
		{num: math.NaN(), filterNums: []float64{}, operator: "exist", want: false},
		{num: math.NaN(), filterNums: []float64{}, operator: "!exist", want: true},

		{num: 10, filterNums: []float64{10}, operator: "=", want: true},
		{num: 10, filterNums: []float64{10, 20}, operator: "=", want: true},
		{num: 10, filterNums: []float64{}, operator: "=", want: false},
		{num: 10, filterNums: []float64{math.NaN()}, operator: "=", want: false},

		{num: 10, filterNums: []float64{5, 10, 15}, operator: ">", want: true},
		{num: 10, filterNums: []float64{}, operator: ">", want: false},
		{num: 10, filterNums: []float64{10}, operator: ">", want: false},
		{num: 10, filterNums: []float64{15}, operator: ">", want: false},

		{num: 10, filterNums: []float64{5, 10, 15}, operator: ">=", want: true},
		{num: 10, filterNums: []float64{}, operator: ">=", want: false},
		{num: 10, filterNums: []float64{10}, operator: ">=", want: true},
		{num: 10, filterNums: []float64{15}, operator: ">=", want: false},

		{num: 10, filterNums: []float64{5, 10, 15}, operator: "<", want: true},
		{num: 10, filterNums: []float64{}, operator: "<", want: false},
		{num: 10, filterNums: []float64{10}, operator: "<", want: false},
		{num: 10, filterNums: []float64{15}, operator: "<", want: true},

		{num: 10, filterNums: []float64{5, 10, 15}, operator: "<=", want: true},
		{num: 10, filterNums: []float64{}, operator: "<=", want: false},
		{num: 10, filterNums: []float64{10}, operator: "<=", want: true},
		{num: 10, filterNums: []float64{15}, operator: "<=", want: true},

		{num: 10, filterNums: []float64{5, 15}, operator: "!=", want: true},
		{num: 10, filterNums: []float64{}, operator: "!=", want: false},
		{num: 10, filterNums: []float64{math.NaN()}, operator: "!=", want: false},

		{num: 10, filterNums: []float64{}, operator: "fakeop", want: false},
		{num: 10, filterNums: []float64{10}, operator: "fakeop", want: false},
	}

	for _, tc := range testCases {
		got := _checkNumberFilter(tc.num, tc.filterNums, tc.operator)
		if got != tc.want {
			t.Errorf("_checkNumberFilter(%v, %v, %s) = %v; want %v", tc.num, tc.filterNums, tc.operator, got, tc.want)
		}
	}
}

func TestCheckValueExists(t *testing.T) {
	type ValueTestCase struct {
		value interface{}
		want  bool
	}

	testCases := []ValueTestCase{
		{value: "test", want: true},
		{value: 123, want: true},
		{value: true, want: true},
		{value: 1.23, want: true},
		{value: nil, want: false},
		{value: "", want: false},
		{value: math.NaN(), want: false},
		{value: struct{}{}, want: false},
	}

	for _, tc := range testCases {
		got := checkValueExists(tc.value)
		if got != tc.want {
			t.Errorf("checkValueExists(%v) = %v; want %v", tc.value, got, tc.want)
		}
	}
}
func TestDoesUserPassFilter_WithUserIDFilter(t *testing.T) {
	user := api.PopulatedUser{
		User: api.User{
			UserId: "1234",
		},
		PlatformData: (&api.PlatformData{}).Default(),
	}

	testCases := []struct {
		name       string
		comparator string
		values     []interface{}
		expected   bool
	}{
		{
			name:       "User id equals filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"1234"},
			expected:   true,
		},
		{
			name:       "User id does not equal filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"5678"},
			expected:   false,
		},
		{
			name:       "User id in filter set",
			comparator: ComparatorContain,
			values:     []interface{}{"5678", "1234", "000099"},
			expected:   true,
		},
	}

	for _, tc := range testCases {
		testFilter := &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "user_id",
				Comparator: tc.comparator,
			},
			Values: tc.values,
		}
		require.NoError(t, testFilter.Initialize())
		result := testFilter.Evaluate(nil, user, nil)
		if result != tc.expected {
			t.Errorf("doesUserPassFilter(%v) = %v; want %v", tc.name, result, tc.expected)
		}
	}
}

func TestDoesUserPassFilter_WithUserCountryFilter(t *testing.T) {
	user := api.PopulatedUser{
		User: api.User{
			UserId:  "1234",
			Country: "CA",
		},
		PlatformData: (&api.PlatformData{}).Default(),
	}

	testCases := []struct {
		name       string
		comparator string
		values     []interface{}
		expected   bool
	}{
		{
			name:       "User country equals filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"CA"},
			expected:   true,
		},
		{
			name:       "User country does not equal filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"JP"},
			expected:   false,
		},
		{
			name:       "User country in filter set",
			comparator: ComparatorContain,
			values:     []interface{}{"US", "JP", "CA"},
			expected:   true,
		},
	}

	for _, tc := range testCases {
		testFilter := &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "country",
				Comparator: tc.comparator,
			},
			Values: tc.values,
		}
		require.NoError(t, testFilter.Initialize())
		result := testFilter.Evaluate(nil, user, nil)
		if result != tc.expected {
			t.Errorf("doesUserPassFilter(%v) = %v; want %v", tc.name, result, tc.expected)
		}
	}
}

func TestDoesUserPassFilter_WithUserEmailFilter(t *testing.T) {
	user := api.PopulatedUser{
		User: api.User{
			UserId: "1234",
			Email:  "test@devcycle.com",
		},
		PlatformData: (&api.PlatformData{}).Default(),
	}

	testCases := []struct {
		name       string
		comparator string
		values     []interface{}
		expected   bool
	}{
		{
			name:       "User email equals filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"test@devcycle.com"},
			expected:   true,
		},
		{
			name:       "User email does not equal filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"someone.else@devcycle.com"},
			expected:   false,
		},
		{
			name:       "User email in filter set",
			comparator: ComparatorContain,
			values:     []interface{}{"@gmail.com", "@devcycle.com", "@hotmail.com"},
			expected:   true,
		},
	}

	for _, tc := range testCases {
		testFilter := &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "email",
				Comparator: tc.comparator,
			},
			Values: tc.values,
		}
		require.NoError(t, testFilter.Initialize())
		result := testFilter.Evaluate(nil, user, nil)
		if result != tc.expected {
			t.Errorf("doesUserPassFilter(%v) = %v; want %v", tc.name, result, tc.expected)
		}
	}
}

func TestDoesUserPassFilter_WithUserAppVersionFilter(t *testing.T) {
	user := api.PopulatedUser{
		User: api.User{
			UserId:     "1234",
			AppVersion: "1.2.3",
		},
		PlatformData: (&api.PlatformData{}).Default(),
	}

	testCases := []struct {
		name       string
		comparator string
		values     []interface{}
		expected   bool
	}{
		{
			name:       "User app version equals filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"1.2.3"},
			expected:   true,
		},
		{
			name:       "User app version does not equal filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"0.0.1"},
			expected:   false,
		},
	}

	for _, tc := range testCases {
		testFilter := &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "appVersion",
				Comparator: tc.comparator,
			},
			Values: tc.values,
		}
		require.NoError(t, testFilter.Initialize())
		result := testFilter.Evaluate(nil, user, nil)
		if result != tc.expected {
			t.Errorf("doesUserPassFilter(%v) = %v; want %v", tc.name, result, tc.expected)
		}
	}
}

func TestDoesUserPassFilter_WithUserPlatformVersionFilter(t *testing.T) {
	user := api.PopulatedUser{
		User: api.User{
			UserId: "1234",
		},
		PlatformData: &api.PlatformData{
			Platform:        "iOS",
			PlatformVersion: "10.3.1",
		},
	}

	testCases := []struct {
		name       string
		comparator string
		values     []interface{}
		expected   bool
	}{
		{
			name:       "User platform version equals filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"10.3.1"},
			expected:   true,
		},
		{
			name:       "User platform version does not equal filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"0.0.1"},
			expected:   false,
		},
		{
			name:       "User platform version is greater",
			comparator: ComparatorGreaterEqual,
			values:     []interface{}{"10.3"},
			expected:   true,
		},
	}

	for _, tc := range testCases {
		testFilter := &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "platformVersion",
				Comparator: tc.comparator,
			},
			Values: tc.values,
		}
		require.NoError(t, testFilter.Initialize())
		result := testFilter.Evaluate(nil, user, nil)
		if result != tc.expected {
			t.Errorf("doesUserPassFilter(%v) = %v; want %v", tc.name, result, tc.expected)
		}
	}
}

func TestDoesUserPassFilter_WithUserDeviceModelFilter(t *testing.T) {
	user := api.PopulatedUser{
		User: api.User{
			UserId:      "1234",
			DeviceModel: "Samsung Galaxy F12",
		},
		PlatformData: (&api.PlatformData{}).Default(),
	}

	testCases := []struct {
		name       string
		comparator string
		values     []interface{}
		expected   bool
	}{
		{
			name:       "User device model equals filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"Samsung Galaxy F12"},
			expected:   true,
		},
		{
			name:       "User device model does not equal filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"iPhone X"},
			expected:   false,
		},
		{
			name:       "User device model in filter set",
			comparator: ComparatorContain,
			values:     []interface{}{"iPhone X", "Google Pixel 49", "Samsung Galaxy F12"},
			expected:   true,
		},
	}

	for _, tc := range testCases {
		testFilter := &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "deviceModel",
				Comparator: tc.comparator,
			},
			Values: tc.values,
		}
		require.NoError(t, testFilter.Initialize())
		result := testFilter.Evaluate(nil, user, nil)
		if result != tc.expected {
			t.Errorf("doesUserPassFilter(%v) = %v; want %v", tc.name, result, tc.expected)
		}
	}
}

func TestDoesUserPassFilter_WithUserPlatformFilter(t *testing.T) {
	user := api.PopulatedUser{
		User: api.User{
			UserId: "1234",
		},
		PlatformData: &api.PlatformData{
			Platform:        "iOS",
			PlatformVersion: "10.3.1",
		},
	}

	testCases := []struct {
		name       string
		comparator string
		values     []interface{}
		expected   bool
	}{
		{
			name:       "User platform equals filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"iOS"},
			expected:   true,
		},
		{
			name:       "User platform does not equal filter",
			comparator: ComparatorEqual,
			values:     []interface{}{"Linux"},
			expected:   false,
		},
		{
			name:       "User platform in filter set",
			comparator: ComparatorContain,
			values:     []interface{}{"Linux", "macOS", "iOS"},
			expected:   true,
		},
	}

	for _, tc := range testCases {
		testFilter := &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "platform",
				Comparator: tc.comparator,
			},
			Values: tc.values,
		}
		require.NoError(t, testFilter.Initialize())
		result := testFilter.Evaluate(nil, user, nil)
		if result != tc.expected {
			t.Errorf("doesUserPassFilter(%v) = %v; want %v", tc.name, result, tc.expected)
		}
	}
}
