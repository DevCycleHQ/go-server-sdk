package bucketing

import "github.com/devcyclehq/go-server-sdk/v2/api"

type AudienceOperator struct {
	Operator string       `json:"operator"`
	Filters  MixedFilters `json:"filters"`
}

func (o AudienceOperator) GetOperator() string {
	return o.Operator
}

func (o AudienceOperator) GetFilters() []FilterOrOperator {
	return o.Filters
}

func (operator AudienceOperator) Evaluate(audiences map[string]NoIdAudience, user api.PopulatedUser, clientCustomData map[string]interface{}) bool {
	if len(operator.GetFilters()) == 0 {
		return false
	}
	if operator.GetOperator() == OperatorOr {
		for _, filter := range operator.GetFilters() {
			if filter.Evaluate(audiences, user, clientCustomData) {
				return true
			}
		}
		return false
	} else if operator.GetOperator() == OperatorAnd {
		for _, filter := range operator.GetFilters() {
			if !filter.Evaluate(audiences, user, clientCustomData) {
				return false
			}
		}
		return true
	}
	return false
}
