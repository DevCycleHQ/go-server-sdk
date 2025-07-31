package api

type EvaluationReason string
type DefaultReason string

const (
	DefaultReasonMissingConfig               DefaultReason = "Missing Config"
	DefaultReasonMissingVariable             DefaultReason = "Missing Variable"
	DefaultReasonMissingFeature              DefaultReason = "Missing Feature"
	DefaultReasonMissingVariation            DefaultReason = "Missing Variation"
	DefaultReasonMissingVariableForVariation DefaultReason = "Missing Variable for Variation"
	DefaultReasonUserNotInRollout            DefaultReason = "User Not in Rollout"
	DefaultReasonUserNotTargeted             DefaultReason = "User Not Targeted"
	DefaultReasonInvalidVariableType         DefaultReason = "Invalid Variable Type"
	DefaultReasonVariableTypeMismatch        DefaultReason = "Variable Type Mismatch"
	DefaultReasonUnknown                     DefaultReason = "Unknown"
	DefaultReasonError                       DefaultReason = "Error"
	DefaultReasonNotDefaulted                DefaultReason = ""
)

const (
	EvaluationReasonTargetingMatch EvaluationReason = "TARGETING_MATCH"
	EvaluationReasonSplit          EvaluationReason = "SPLIT"
	EvaluationReasonDefault        EvaluationReason = "DEFAULT"
	EvaluationReasonDisabled       EvaluationReason = "DISABLED"
	EvaluationReasonError          EvaluationReason = "ERROR"
)
