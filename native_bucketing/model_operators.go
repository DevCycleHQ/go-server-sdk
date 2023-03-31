package native_bucketing

type BaseOperator interface {
	Operator() string
	Filters() []FilterOrOperator
}

type TopLevelOperator struct {
	BaseOperator
	operator string
	filters  []FilterOrOperator
}

func (t TopLevelOperator) Operator() string {
	return t.operator
}

func (t TopLevelOperator) Filters() []FilterOrOperator {
	return t.filters
}

type AudienceOperator struct {
	BaseOperator
	operator string
	filters  []FilterOrOperator
}

func (t AudienceOperator) Operator() string {
	return t.operator
}

func (t AudienceOperator) Filters() []FilterOrOperator {
	return t.filters
}
