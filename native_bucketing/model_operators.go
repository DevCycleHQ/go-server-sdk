package native_bucketing

type BaseOperator interface {
	GetOperator() string
	GetFilters() []BaseFilter
}

type AudienceOperator struct {
	BaseOperator
	Operator string       `json:"operator"`
	Filters  MixedFilters `json:"filters"`
}

func (o AudienceOperator) GetOperator() string {
	return o.Operator
}

func (o AudienceOperator) GetFilters() []BaseFilter {
	return o.Filters
}
