package bucketing

import (
	_ "embed"
)

const (
	OperatorAnd = "and"
	OperatorOr  = "or"
)

const (
	VariableEvaluatedEvent    = "variableEvaluated"
	VariableDefaultedEvent    = "variableDefaulted"
	AggVariableEvaluatedEvent = "aggVariableEvaluated"
	AggVariableDefaultedEvent = "aggVariableDefaulted"
)

const (
	TypeAll           = "all"
	TypeUser          = "user"
	TypeOptIn         = "optIn"
	TypeAudienceMatch = "audienceMatch"
)

const (
	SubTypeUserID          = "user_id"
	SubTypeEmail           = "email"
	SubTypeIP              = "ip"
	SubTypeCountry         = "country"
	SubTypePlatform        = "platform"
	SubTypePlatformVersion = "platformVersion"
	SubTypeAppVersion      = "appVersion"
	SubTypeDeviceModel     = "deviceModel"
	SubTypeCustomData      = "customData"
)

const (
	ComparatorEqual        = "="
	ComparatorNotEqual     = "!="
	ComparatorGreater      = ">"
	ComparatorGreaterEqual = ">="
	ComparatorLess         = "<"
	ComparatorLessEqual    = "<="
	ComparatorExist        = "exist"
	ComparatorNotExist     = "!exist"
	ComparatorContain      = "contain"
	ComparatorNotContain   = "!contain"
	ComparatorStartWith    = "startWith"
	ComparatorNotStartWith = "!startWith"
	ComparatorEndWith      = "endWith"
	ComparatorNotEndWith   = "!endWith"
)

const (
	DataKeyTypeString  = "String"
	DataKeyTypeBoolean = "Boolean"
	DataKeyTypeNumber  = "Number"
)

const (
	VariableTypesString = "String"
	VariableTypesNumber = "Number"
	VariableTypesJSON   = "JSON"
	VariableTypesBool   = "Boolean"
)
