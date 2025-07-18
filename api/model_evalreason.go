package api

type EvaluationReason string
type DefaultReason string

const (
	DefaultReasonMissingConfig               DefaultReason = "MISSING_CONFIG"
	DefaultReasonMissingVariable             DefaultReason = "MISSING_VARIABLE"
	DefaultReasonMissingFeature              DefaultReason = "MISSING_FEATURE"
	DefaultReasonMissingVariation            DefaultReason = "MISSING_VARIATION"
	DefaultReasonMissingVariableForVariation DefaultReason = "MISSING_VARIABLE_FOR_VARIATION"
	DefaultReasonUserNotInRollout            DefaultReason = "USER_NOT_IN_ROLLOUT"
	DefaultReasonUserNotTargeted             DefaultReason = "USER_NOT_TARGETED"
	DefaultReasonInvalidVariableType         DefaultReason = "INVALID_VARIABLE_TYPE"
	DefaultReasonUnknown                     DefaultReason = "UNKNOWN"
)

const (
	EvaluationReasonTargetingMatch EvaluationReason = "TARGETING_MATCH"
	EvaluationReasonSplit          EvaluationReason = "SPLIT"
	EvaluationReasonDefault        EvaluationReason = "DEFAULT"
	EvaluationReasonDisabled       EvaluationReason = "DISABLED"
	EvaluationReasonError          EvaluationReason = "ERROR"
)
