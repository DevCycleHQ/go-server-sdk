package bucketing

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/devcyclehq/go-server-sdk/v2/api"
)

// benchCustomDataConfig returns a config with one feature. The feature has one
// customData target for each site ID, then a target for all users.
func benchCustomDataConfig(b *testing.B, targets int) []byte {
	b.Helper()
	customDataTarget := func(i int) map[string]any {
		return map[string]any{
			"_id": fmt.Sprintf("target-%d", i),
			"_audience": map[string]any{
				"_id": fmt.Sprintf("audience-%d", i),
				"filters": map[string]any{
					"operator": "and",
					"filters": []any{map[string]any{
						"type": "user", "subType": "customData", "dataKey": "site_id",
						"dataKeyType": "String", "comparator": "=", "values": []any{fmt.Sprintf("site-%d", i)},
					}},
				},
			},
			"distribution": []any{map[string]any{"_variation": "variation-on", "percentage": 1}},
		}
	}
	list := make([]any, 0, targets+1)
	for i := range targets {
		list = append(list, customDataTarget(i))
	}
	list = append(list, map[string]any{
		"_id": "target-all",
		"_audience": map[string]any{
			"_id":     "audience-all",
			"filters": map[string]any{"operator": "and", "filters": []any{map[string]any{"type": "all"}}},
		},
		"distribution": []any{map[string]any{"_variation": "variation-off", "percentage": 1}},
	})
	variation := func(id string, value bool) map[string]any {
		return map[string]any{
			"_id": id, "name": id, "key": id,
			"variables": []any{map[string]any{"_var": "bench-var-id", "value": value}},
		}
	}
	config := map[string]any{
		"project":     map[string]any{"_id": "bench-project", "key": "bench-project", "a0_organization": "org_bench", "settings": map[string]any{}},
		"environment": map[string]any{"_id": "bench-environment", "key": "bench-environment"},
		"audiences":   map[string]any{},
		"features": []any{map[string]any{
			"_id": "bench-feature", "type": "release", "key": "bench-feature",
			"configuration": map[string]any{"_id": "bench-configuration", "targets": list},
			"variations":    []any{variation("variation-on", true), variation("variation-off", false)},
		}},
		"variables": []any{map[string]any{"_id": "bench-var-id", "type": "Boolean", "key": "bench-var"}},
	}
	body, err := json.Marshal(config)
	if err != nil {
		b.Fatal(err)
	}
	return body
}

func BenchmarkVariableForUser_CustomDataTargets(b *testing.B) {
	// Three keys fit in a map on the stack. Twelve keys need a map on the heap.
	small := map[string]interface{}{
		"site_id":    "site-unmatched",
		"account_id": "account",
		"groups":     "pro",
	}
	large := map[string]interface{}{
		"site_id":         "site-unmatched",
		"account_id":      "account",
		"groups":          "pro",
		"path":            "/index.html",
		"ja4_fingerprint": "t13d1516h2_8daaf6152771_02713d6af862",
		"user_agent":      "Mozilla/5.0",
		"query":           "",
		"referrer":        "",
		"hostname":        "example.netlify.app",
		"cdn_group":       "global",
		"address":         "192.0.2.1",
		"address_int":     float64(3221225985),
	}
	for _, targets := range []int{1, 37} {
		for _, customData := range []map[string]interface{}{small, large} {
			b.Run(fmt.Sprintf("targets=%d/keys=%d", targets, len(customData)), func(b *testing.B) {
				if err := SetConfig(benchCustomDataConfig(b, targets), "bench", "", "", ""); err != nil {
					b.Fatal(err)
				}
				user := api.User{
					UserId:     "example.netlify.app",
					CustomData: customData,
				}.GetPopulatedUser(&api.PlatformData{})

				_, value, _, _, _, err := generateBucketedVariableForUser("bench", user, "bench-var", nil)
				if err != nil || value != false {
					b.Fatalf("value = %v, err = %v", value, err)
				}

				b.ReportAllocs()
				for b.Loop() {
					_, _, _, _, _, _ = generateBucketedVariableForUser("bench", user, "bench-var", nil)
				}
			})
		}
	}
}
