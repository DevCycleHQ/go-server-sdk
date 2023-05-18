package bucketing

import (
	"github.com/devcyclehq/go-server-sdk/v2/api"
)

type FeatureConfiguration struct {
	Id               string                 `json:"_id"`
	Prerequisites    []FeaturePrerequisites `json:"prerequisites"`
	WinningVariation api.FeatureVariation   `json:"winningVariation"`
	ForcedUsers      map[string]string      `json:"forcedUsers"`
	Targets          []*Target              `json:"targets"`
}

type FeaturePrerequisites struct {
	Feature    string `json:"_feature"`
	Comparator string `json:"comparator" validate:"regexp=^(=|!=)$"`
}
