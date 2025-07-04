package bucketing

import (
	"time"
)

type Target struct {
	Id           string               `json:"_id"`
	Audience     *Audience            `json:"_audience"`
	Rollout      *Rollout             `json:"rollout"`
	Distribution []TargetDistribution `json:"distribution"`
	BucketingKey string               `json:"bucketingKey"`
}

func (t *Target) DecideTargetVariation(boundedHash float64) (string, error) {
	var distributionIndex float64 = 0
	var previousDistributionIndex float64 = 0

	for _, d := range t.Distribution {
		distributionIndex += d.Percentage
		if boundedHash >= previousDistributionIndex && (boundedHash < distributionIndex || (distributionIndex == 1 && boundedHash == 1)) {
			return d.Variation, nil
		}
	}
	return "", ErrFailedToDecideVariation
}

type NoIdAudience struct {
	Filters *AudienceOperator `json:"filters"`
}

type Audience struct {
	NoIdAudience
	Id string `json:"_id"`
}

type Rollout struct {
	Type            string         `json:"type" validate:"regexp=^(schedule|gradual|stepped)$"`
	StartPercentage float64        `json:"startPercentage"`
	StartDate       time.Time      `json:"startDate"`
	Stages          []RolloutStage `json:"stages"`
}

type RolloutStage struct {
	Type       string    `json:"type"`
	Date       time.Time `json:"date"`
	Percentage float64   `json:"percentage" validate:"regexp=^(linear|discrete)$"`
}

type TargetDistribution struct {
	Variation  string  `json:"_variation"`
	Percentage float64 `json:"percentage"`
}
