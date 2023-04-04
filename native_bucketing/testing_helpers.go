package native_bucketing

func checkNumberFilter(number float64, filter *UserFilter) bool {
	operator := filter.GetComparator()
	values := getFilterValuesAsF64(filter)
	return _checkNumberFilter(number, values, operator)
}
