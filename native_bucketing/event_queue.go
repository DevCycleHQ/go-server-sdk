package native_bucketing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
)

type aggEventData struct {
	event                *api.DVCEvent
	variableVariationMap map[string]FeatureVariation
	aggregateByVariation bool
}

type userEventData struct {
	event *api.DVCEvent
	user  *DVCUser
}

type VariationAggMap map[string]int64
type FeatureAggMap map[string]VariationAggMap
type VariableAggMap map[string]FeatureAggMap

type AggregateEventQueue map[string]VariableAggMap
type UserEventQueue map[string]api.UserEventsBatchRecord

func (u *UserEventQueue) BuildBatchRecords() []api.UserEventsBatchRecord {
	var records []api.UserEventsBatchRecord
	for _, record := range *u {
		records = append(records, record)
	}
	return records
}

func (agg *AggregateEventQueue) BuildBatchRecords() api.UserEventsBatchRecord {
	var aggregateEvents []api.DVCEvent
	userId, err := os.Hostname()
	if err != nil {
		userId = "aggregate"
	}

	for _type, variableAggMap := range *agg {
		for variableKey, featureAggMap := range variableAggMap {

			if variationAggMap, ok := featureAggMap["value"]; ok {
				if variationValue, ok := variationAggMap["value"]; ok {
					value := float64(variationValue)
					event := api.DVCEvent{
						Type_:  _type,
						Target: variableKey,
						Value:  value,
						UserId: userId,
					}
					aggregateEvents = append(aggregateEvents, event)
				}
			} else {
				for feature, _variationAggMap := range featureAggMap {
					for variation, count := range _variationAggMap {
						event := api.DVCEvent{
							Type_:  _type,
							Target: variableKey,
							Value:  float64(count),
							UserId: userId,
							MetaData: map[string]interface{}{
								"_variation": variation,
								"_feature":   feature,
							},
						}
						aggregateEvents = append(aggregateEvents, event)
					}
				}
			}
		}
	}
	user := api.DVCUser{UserId: userId}.GetPopulatedUser(platformData)
	return api.UserEventsBatchRecord{
		User:   user,
		Events: aggregateEvents,
	}
}

func InitEventQueue(sdkKey string, options *api.EventQueueOptions) (*EventQueue, error) {
	if sdkKey == "" {
		return nil, fmt.Errorf("sdk key is required")
	}

	options.CheckBounds()
	ctx, cancel := context.WithCancel(context.TODO())

	eq := &EventQueue{
		sdkKey:            sdkKey,
		options:           options,
		aggEventQueueRaw:  make(chan aggEventData, options.MaxEventQueueSize),
		userEventQueueRaw: make(chan userEventData, options.MaxEventQueueSize),
		userEventQueue:    make(map[string]api.UserEventsBatchRecord),
		aggEventQueue:     make(AggregateEventQueue),
		aggEventMutex:     &sync.RWMutex{},
		httpClient:        &http.Client{},
		flushMutex:        &sync.Mutex{},
		pendingPayloads:   make(map[string]api.FlushPayload, 0),
		done:              cancel,
	}

	go eq.processEvents(ctx)

	return eq, nil
}

type EventQueue struct {
	sdkKey            string
	options           *api.EventQueueOptions
	aggEventQueueRaw  chan aggEventData
	userEventQueueRaw chan userEventData
	userEventQueue    UserEventQueue
	aggEventQueue     AggregateEventQueue
	aggEventMutex     *sync.RWMutex
	eventsFlushed     atomic.Int32
	eventsReported    atomic.Int32
	httpClient        *http.Client
	flushMutex        *sync.Mutex
	pendingPayloads   map[string]api.FlushPayload
	done              func()
}

func (eq *EventQueue) MergeAggEventQueueKeys(config *configBody) {
	if eq.aggEventQueue == nil {
		eq.aggEventQueue = make(AggregateEventQueue)
	}
	eq.aggEventMutex.Lock()
	defer eq.aggEventMutex.Unlock()
	for _, target := range []string{api.EventType_AggVariableEvaluated, api.EventType_AggVariableDefaulted, api.EventType_VariableEvaluated, api.EventType_VariableDefaulted} {
		if _, ok := eq.aggEventQueue[target]; !ok {
			eq.aggEventQueue[target] = make(VariableAggMap, len(config.Variables))
		}
		for _, variable := range config.Variables {
			if _, ok := eq.aggEventQueue[target][variable.Key]; !ok {
				eq.aggEventQueue[target][variable.Key] = make(FeatureAggMap, len(config.Features))
			}
			for _, feature := range config.Features {
				if _, ok := eq.aggEventQueue[target][variable.Key][feature.Key]; !ok {
					eq.aggEventQueue[target][variable.Key][feature.Key] = make(VariationAggMap, len(feature.Variations))
				}
				for _, variation := range feature.Variations {
					if _, ok := eq.aggEventQueue[target][variable.Key][feature.Key][variation.Key]; !ok {
						eq.aggEventQueue[target][variable.Key][feature.Key][variation.Key] = 0
					}
				}
			}
		}
	}
}

// QueueAggregateEvent queues an aggregate event to be sent to the server - but offloads actual computing of the event itself
// to a different goroutine.
func (eq *EventQueue) QueueAggregateEvent(config BucketedUserConfig, event api.DVCEvent) error {
	if eq.options != nil && eq.options.IsEventLoggingDisabled(&event) {
		return nil
	}

	if event.Target == "" {
		return fmt.Errorf("target is required for aggregate events")
	}

	select {
	case eq.aggEventQueueRaw <- aggEventData{
		event:                &event,
		variableVariationMap: config.VariableVariationMap,
		aggregateByVariation: event.Type_ == api.EventType_AggVariableEvaluated,
	}:
		util.Debugf("Queued event: %+v", event)
	default:
		return fmt.Errorf("event queue is full, dropping event: %+v", event)
	}

	return nil
}

func (eq *EventQueue) QueueEvent(user DVCUser, event api.DVCEvent) error {

	select {
	case eq.userEventQueueRaw <- userEventData{
		event: &event,
		user:  &user,
	}:
		util.Debugf("Queued event: %+v", event)
	default:
		return util.Errorf("event queue is full, dropping event: %+v", event)
	}

	return nil
}

func (eq *EventQueue) flushEventQueue() (map[string]api.FlushPayload, error) {
	var records []api.UserEventsBatchRecord

	for _, record := range eq.pendingPayloads {
		if record.Status == "failed" {
			return nil, util.Errorf("Cannot flush events, event queue has failed payloads")
		}
	}

	records = append(records, eq.aggEventQueue.BuildBatchRecords())
	records = append(records, eq.userEventQueue.BuildBatchRecords()...)

	for _, record := range records {
		var payload *api.FlushPayload
		for _, pl := range eq.pendingPayloads {
			if pl.Status == "failed" {
				continue
			}
			if pl.EventCount < eq.options.EventRequestChunkSize {
				payload = &pl
			}
		}
		if payload == nil {
			payload = &api.FlushPayload{
				PayloadId: uuid.New().String(),
			}
		}
		payload.AddBatchRecordForUser(record, eq.options.EventRequestChunkSize)
		eq.pendingPayloads[payload.PayloadId] = *payload
	}
	eq.updateFailedPayloads()

	return eq.pendingPayloads, nil
}

func (eq *EventQueue) FlushEvents() (err error) {

	eq.flushMutex.Lock()
	defer eq.flushMutex.Unlock()
	util.Debugf("Started flushing events")

	payloads, err := eq.flushEventQueue()
	if err != nil {
		return err
	}

	for _, payload := range payloads {
		_ = eq.flushEventPayload(&payload)
	}

	util.Debugf("Finished flushing events")

	return
}

func (eq *EventQueue) flushEventPayload(payload *api.FlushPayload) error {
	eventsHost := eq.options.EventsAPIBasePath
	var req *http.Request
	var resp *http.Response
	requestBody, err := json.Marshal(api.BatchEventsBody{Batch: payload.Records})
	if err != nil {
		_ = util.Errorf("Failed to marshal batch events body: %s", err)
		_ = eq.reportPayloadFailure(payload, false)
		return err
	}
	req, err = http.NewRequest("POST", eventsHost+"/v1/events/batch", bytes.NewReader(requestBody))
	if err != nil {
		_ = util.Errorf("Failed to create request to events api: %s", err)
		_ = eq.reportPayloadFailure(payload, false)
		return err
	}

	req.Header.Set("Authorization", eq.sdkKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err = eq.httpClient.Do(req)

	if err != nil {
		_ = util.Errorf("Failed to make request to events api: %s", err)
		_ = eq.reportPayloadFailure(payload, false)
		return err
	}

	// always ensure body is closed to avoid goroutine leak
	defer func() {
		_ = resp.Body.Close()
	}()

	// always read response body fully - from net/http docs:
	// If the Body is not both read to EOF and closed, the Client's
	// underlying RoundTripper (typically Transport) may not be able to
	// re-use a persistent TCP connection to the server for a subsequent
	// "keep-alive" request.
	responseBody, readError := io.ReadAll(resp.Body)
	if readError != nil {
		_ = util.Errorf("Failed to read response body: %v", readError)
		_ = eq.reportPayloadFailure(payload, false)
		return err
	}

	if resp.StatusCode >= 500 {
		util.Warnf("Events API Returned a 5xx error, retrying later.")
		_ = eq.reportPayloadFailure(payload, true)
		return err
	}

	if resp.StatusCode >= 400 {
		_ = eq.reportPayloadFailure(payload, false)
		_ = util.Errorf("Error sending events - Response: %s", string(responseBody))
		return err
	}

	if resp.StatusCode == 201 {
		err = eq.reportPayloadSuccess(payload)
		eq.eventsReported.Add(1)
		return err
	}

	_ = util.Errorf("unknown status code when flushing events %d", resp.StatusCode)
	return eq.reportPayloadFailure(payload, false)
}

func (eq *EventQueue) Metrics() (int32, int32) {
	return eq.eventsFlushed.Load(), eq.eventsReported.Load()
}

func (eq *EventQueue) Close() (err error) {
	err = eq.FlushEvents()
	eq.done()
	return
}

func (eq *EventQueue) updateFailedPayloads() {
	eq.flushMutex.Lock()
	defer eq.flushMutex.Unlock()
	for _, pl := range eq.pendingPayloads {
		if pl.Status == "failed" {
			pl.Status = "sending"
		}
	}
}

func (eq *EventQueue) reportPayloadSuccess(payload *api.FlushPayload) error {
	eq.flushMutex.Lock()
	defer eq.flushMutex.Unlock()
	if _, ok := eq.pendingPayloads[payload.PayloadId]; ok {
		delete(eq.pendingPayloads, payload.PayloadId)
	} else {
		return util.Errorf("Failed to find payload: %s to mark as success", payload.PayloadId)
	}
	return nil
}

func (eq *EventQueue) reportPayloadFailure(payload *api.FlushPayload, retryable bool) error {
	eq.flushMutex.Lock()
	defer eq.flushMutex.Unlock()

	if v, ok := eq.pendingPayloads[payload.PayloadId]; ok {
		if retryable {
			v.Status = "failed"
		} else {
			delete(eq.pendingPayloads, payload.PayloadId)
		}
	} else {
		return util.Errorf("Failed to find payload: %s, retryable: %b", payload.PayloadId, retryable)
	}
	return nil
}

func (eq *EventQueue) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Closing native event queues")
			close(eq.userEventQueueRaw)
			close(eq.aggEventQueueRaw)
			return
		case userEvent := <-eq.userEventQueueRaw:
			err := eq.processUserEvent(userEvent)
			if err != nil {
				return
			}
		case aggEvent := <-eq.aggEventQueueRaw:
			err := eq.processAggregateEvent(aggEvent)
			if err != nil {
				return
			}
		}
	}
}

func (eq *EventQueue) processUserEvent(event userEventData) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic in processUserEvent: %v", r)
			if errVal, ok := r.(error); ok {
				err = errVal
			}
		}
	}()
	// TODO: provide platform data
	popU := event.user.GetPopulatedUser(platformData)
	ccd := GetClientCustomData(eq.sdkKey)
	popU.MergeClientCustomData(ccd)

	bucketedConfig, err := GenerateBucketedConfig(eq.sdkKey, popU, ccd)
	if err != nil {
		// TODO: Log
		return err
	}
	event.event.FeatureVars = bucketedConfig.FeatureVariationMap
	if _, ok := eq.userEventQueue[popU.UserId]; ok {
		records := eq.userEventQueue[popU.UserId]
		records.Events = append(records.Events, *event.event)
		records.User = popU
		eq.userEventQueue[popU.UserId] = records
	} else {
		record := api.UserEventsBatchRecord{
			User:   popU,
			Events: []api.DVCEvent{*event.event},
		}
		eq.userEventQueue[popU.UserId] = record
	}
	return nil
}

func (eq *EventQueue) processAggregateEvent(event aggEventData) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic in processAggregateEvent: %v", r)
			if errVal, ok := r.(error); ok {
				err = errVal
			}
		}
	}()

	eq.aggEventMutex.Lock()
	defer eq.aggEventMutex.Unlock()
	eType := event.event.Type_
	eTarget := event.event.Target

	variableFeatureVariationAggregationMap := make(VariableAggMap)
	if v, ok := eq.aggEventQueue[eType]; ok {
		variableFeatureVariationAggregationMap = v
	} else {
		eq.aggEventQueue[eType] = variableFeatureVariationAggregationMap
	}
	featureVariationAggregationMap := make(FeatureAggMap)
	if v, ok := variableFeatureVariationAggregationMap[eTarget]; ok {
		featureVariationAggregationMap = v
	} else {
		variableFeatureVariationAggregationMap[eTarget] = featureVariationAggregationMap
	}

	if event.aggregateByVariation {
		if _, ok := event.variableVariationMap[eTarget]; !ok {
			return fmt.Errorf("target mapping not found in variableVariationMap for %s", eTarget)
		}
		featureVar := event.variableVariationMap[eTarget]
		variationAggMap := make(VariationAggMap)
		if v, ok := featureVariationAggregationMap[featureVar.Feature]; ok {
			variationAggMap = v
		}
		variationAggMap[featureVar.Variation] += 1
		featureVariationAggregationMap[featureVar.Feature] = variationAggMap
	} else {
		if feature, ok := featureVariationAggregationMap["value"]; ok {
			if _, ok2 := feature["value"]; ok2 {
				feature["value"]++
			} else {
				return fmt.Errorf("missing second value map for aggVariableDefaulted")
			}
		} else {
			featureVariationAggregationMap[eTarget] = VariationAggMap{
				"value": 1,
			}
			// increment event queue count
		}
	}
	return nil
}
