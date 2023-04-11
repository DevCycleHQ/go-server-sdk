package native_bucketing

type FeatureConfiguration struct {
	Id               string                 `json:"_id"`
	Prerequisites    []FeaturePrerequisites `json:"prerequisites"`
	WinningVariation FeatureVariation       `json:"winningVariation"`
	ForcedUsers      map[string]string      `json:"forcedUsers"`
	Targets          []Target               `json:"targets"`
}

type FeaturePrerequisites struct {
	Feature    string `json:"_feature"`
	Comparator string `json:"comparator" validate:"regexp=^(=|!=)$"`
}
