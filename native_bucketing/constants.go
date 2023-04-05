package native_bucketing

import (
	_ "embed"
)

var (
	//go:embed testdata/fixture_large_config.json
	test_large_config          string
	test_large_config_variable = "v-key-25"
)

const (
	OperatorAnd = "and"
	OperatorOr  = "or"
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
)

const (
	DataKeyTypeString  = "String"
	DataKeyTypeBoolean = "Boolean"
	DataKeyTypeNumber  = "Number"
)
