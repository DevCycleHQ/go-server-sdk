/*
 * DevCycle Bucketing API
 *
 * Documents the DevCycle Bucketing API which provides and API interface to User Bucketing and for generated SDKs.
 *
 * API version: 1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package devcycle

type Event struct {
	// Custom event type
	Type_ string `json:"type"`
	// Custom event target / subject of event. Contextual to event type
	Target string `json:"target,omitempty"`
	// Unix epoch time the event occurred according to client
	Date float64 `json:"date,omitempty"`
	// Value for numerical events. Contextual to event type
	Value float64 `json:"value,omitempty"`
	// Extra JSON metadata for event. Contextual to event type
	MetaData *interface{} `json:"metaData,omitempty"`
}
