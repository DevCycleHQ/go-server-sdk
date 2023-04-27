/*
 * DevCycle Bucketing API
 *
 * Documents the DevCycle Bucketing API which provides and API interface to User Bucketing and for generated SDKs.
 *
 * API version: 1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package api

import (
	"time"
)

const (
	EventType_VariableEvaluated    = "variableEvaluated"
	EventType_AggVariableEvaluated = "aggVariableEvaluated"
	EventType_VariableDefaulted    = "variableDefaulted"
	EventType_AggVariableDefaulted = "aggVariableDefaulted"
)

type DVCEvent struct {
	Type_       string                 `json:"type"`
	Target      string                 `json:"target,omitempty"`
	CustomType  string                 `json:"customType,omitempty"`
	UserId      string                 `json:"user_id"`
	ClientDate  time.Time              `json:"clientDate"`
	Value       float64                `json:"value,omitempty"`
	FeatureVars map[string]string      `json:"featureVars"`
	MetaData    map[string]interface{} `json:"metaData,omitempty"`
}

type UserEventsBatchRecord struct {
	User   DVCPopulatedUser `json:"user"`
	Events []DVCEvent       `json:"events"`
}

type FlushPayload struct {
	PayloadId  string                  `json:"payloadId"`
	EventCount int                     `json:"eventCount"`
	Records    []UserEventsBatchRecord `json:"records"`
	Status     string
}

func (fp *FlushPayload) AddBatchRecordForUser(record UserEventsBatchRecord, chunkSize int) {
	userRecord := fp.getRecordForUser(record.User.UserId)
	chunkedEvents := ChunkSlice(record.Events, chunkSize)
	if userRecord != nil {
		userRecord.User = record.User
		for _, chunk := range chunkedEvents {
			userRecord.Events = append(userRecord.Events, chunk...)
		}
	} else {
		for _, chunk := range chunkedEvents {
			fp.Records = append(fp.Records, UserEventsBatchRecord{
				User:   record.User,
				Events: chunk,
			})
		}
	}

}

func (fp *FlushPayload) getRecordForUser(userId string) *UserEventsBatchRecord {
	for _, record := range fp.Records {
		if record.User.UserId == userId {
			return &record
		}
	}
	return nil
}

type BatchEventsBody struct {
	Batch []UserEventsBatchRecord `json:"batch"`
}

type EventQueueOptions struct {
	FlushEventsInterval          time.Duration `json:"flushEventsMS"`
	DisableAutomaticEventLogging bool          `json:"disableAutomaticEventLogging"`
	DisableCustomEventLogging    bool          `json:"disableCustomEventLogging"`
	MaxEventQueueSize            int           `json:"maxEventsPerFlush,omitempty"`
	FlushEventQueueSize          int           `json:"minEventsPerFlush,omitempty"`
	EventRequestChunkSize        int           `json:"eventRequestChunkSize,omitempty"`
	EventsAPIBasePath            string        `json:"eventsAPIBasePath,omitempty"`
}

func (o *EventQueueOptions) CheckBounds() {
	if o.MaxEventQueueSize < 100 {
		o.MaxEventQueueSize = 100
	} else if o.MaxEventQueueSize > 1000 {
		o.MaxEventQueueSize = 1000
	}
	if o.EventsAPIBasePath == "" {
		o.EventsAPIBasePath = "https://events.devcycle.com"
	}
}

func (o *EventQueueOptions) IsEventLoggingDisabled(event *DVCEvent) bool {
	switch event.Type_ {
	case EventType_VariableEvaluated, EventType_AggVariableEvaluated, EventType_VariableDefaulted, EventType_AggVariableDefaulted:
		return o.DisableAutomaticEventLogging
	default:
		return o.DisableCustomEventLogging
	}
}
