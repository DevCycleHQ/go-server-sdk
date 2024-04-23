package bucketing

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/google/uuid"
)

var ErrQueueFull = fmt.Errorf("Max queue size reached")

type aggEventData struct {
	eventType     string
	variableKey   string
	featureId     string
	variationId   string
	defaultReason string
}

type userEventData struct {
	event *api.Event
	user  *api.User
}

// Structure of the aggregation maps
// map event type -> event target
// map event target -> feature id
// map feature id -> variation id
// For Evaluation Events:
// ["aggVariableEvaluated"]["somevariablekey"]["feature_id"]["variation_id"] = 1
// For Defaulted Events:
// ["aggVariableDefaulted"]["somevariablekey"]["defaulted"][DEFAULT_REASON] = 1

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

func (agg *AggregateEventQueue) BuildBatchRecords(platformData *api.PlatformData, clientUUID string, configEtag string, rayId string) api.UserEventsBatchRecord {
	var aggregateEvents []api.Event
	userId, err := os.Hostname()
	if err != nil {
		userId = "aggregate"
	}
	emptyFeatureVars := make(map[string]string)

	// type is either aggVariableEvaluated or aggVariableDefaulted
	for _type, variableAggMap := range *agg {
		for variableKey, featureAggMap := range variableAggMap {
			// feature is feature id for evaluation events, or the string "defaulted" for default events
			for feature, _variationAggMap := range featureAggMap {
				// variation is variation id for evaluation events, or the "default reason" for default events
				for variation, count := range _variationAggMap {
					if count == 0 {
						continue
					}
					var metaData map[string]interface{}
					if _type == api.EventType_AggVariableDefaulted {
						metaData = map[string]interface{}{
							"defaultReason": variation,
						}
					} else {
						metaData = map[string]interface{}{
							"_variation": variation,
							"_feature":   feature,
						}
					}

					metaData["clientUUID"] = clientUUID
					if configEtag != "" {
						metaData["configEtag"] = configEtag
					}
					if rayId != "" {
						metaData["configRayId"] = rayId
					}

					event := api.Event{
						Type_:       _type,
						Target:      variableKey,
						Value:       float64(count),
						UserId:      userId,
						MetaData:    metaData,
						FeatureVars: emptyFeatureVars,
						ClientDate:  time.Now(),
					}
					aggregateEvents = append(aggregateEvents, event)
				}
			}
		}
	}
	user := api.User{UserId: userId}.GetPopulatedUser(platformData)
	return api.UserEventsBatchRecord{
		User:   user,
		Events: aggregateEvents,
	}
}

type EventQueue struct {
	sdkKey              string
	options             *api.EventQueueOptions
	aggEventQueueRaw    chan aggEventData
	userEventQueueRaw   chan userEventData
	userEventQueue      UserEventQueue
	userEventQueueCount int
	aggEventQueue       AggregateEventQueue
	stateMutex          *sync.RWMutex
	httpClient          *http.Client
	pendingPayloads     map[string]api.FlushPayload
	done                func()
	eventsFlushed       atomic.Int32
	eventsReported      atomic.Int32
	eventsDropped       atomic.Int32
	platformData        *api.PlatformData
}

func NewEventQueue(sdkKey string, options *api.EventQueueOptions, platformData *api.PlatformData) (*EventQueue, error) {
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
		stateMutex:        &sync.RWMutex{},
		httpClient: &http.Client{
			// Set an explicit timeout so that we don't wait forever on a request
			// Use a separate hardcoded timeout here because event requests should not be blocking.
			Timeout: time.Second * 60,
		},
		pendingPayloads: make(map[string]api.FlushPayload, 0),
		done:            cancel,
		platformData:    platformData,
	}

	if !options.DisableAutomaticEventLogging || !options.DisableCustomEventLogging {
		go eq.processEvents(ctx)
	}

	return eq, nil
}

func (eq *EventQueue) MergeAggEventQueueKeys(config *configBody) {
	eq.stateMutex.Lock()
	defer eq.stateMutex.Unlock()

	if eq.aggEventQueue == nil {
		eq.aggEventQueue = make(AggregateEventQueue)
	}
	for _, target := range []string{api.EventType_AggVariableEvaluated, api.EventType_VariableEvaluated} {
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

func (eq *EventQueue) queueAggregateEventInternal(variableKey, featureId, variationId, eventType string, defaultReason string) error {
	if eq.options != nil && eq.options.IsEventLoggingDisabled(eventType) {
		return nil
	}

	if variableKey == "" {
		return fmt.Errorf("A variable key is required for aggregate events")
	}

	select {
	case eq.aggEventQueueRaw <- aggEventData{
		eventType:     eventType,
		variableKey:   variableKey,
		featureId:     featureId,
		variationId:   variationId,
		defaultReason: defaultReason,
	}:
	default:
		eq.eventsDropped.Add(1)
		return ErrQueueFull
	}

	return nil
}

func (eq *EventQueue) QueueEvent(user api.User, event api.Event) error {

	select {
	case eq.userEventQueueRaw <- userEventData{
		event: &event,
		user:  &user,
	}:
	default:
		eq.eventsDropped.Add(1)
		return ErrQueueFull
	}

	return nil
}

func (eq *EventQueue) QueueVariableEvaluatedEvent(variableKey, featureId, variationId string) error {
	if eq.options.DisableAutomaticEventLogging {
		return nil
	}

	return eq.queueAggregateEventInternal(variableKey, featureId, variationId, api.EventType_AggVariableEvaluated, "")
}

func (eq *EventQueue) QueueVariableDefaultedEvent(variableKey, defaultReason string) error {
	if eq.options.DisableAutomaticEventLogging {
		return nil
	}

	return eq.queueAggregateEventInternal(variableKey, "", "", api.EventType_AggVariableDefaulted, defaultReason)
}

func (eq *EventQueue) FlushEventQueue(clientUUID string, configEtag string, rayId string) (map[string]api.FlushPayload, error) {
	eq.stateMutex.Lock()
	defer eq.stateMutex.Unlock()

	var records []api.UserEventsBatchRecord

	records = append(records, eq.aggEventQueue.BuildBatchRecords(eq.platformData, clientUUID, configEtag, rayId))
	records = append(records, eq.userEventQueue.BuildBatchRecords()...)
	eq.aggEventQueue = make(AggregateEventQueue)
	eq.userEventQueue = make(UserEventQueue)
	eq.userEventQueueCount = 0

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
		payload.EventCount = len(payload.Records)
		if payload.EventCount == 0 {
			continue
		}
		eq.pendingPayloads[payload.PayloadId] = *payload
	}

	eq.updateFailedPayloads()

	eq.eventsFlushed.Add(int32(len(eq.pendingPayloads)))

	return eq.pendingPayloads, nil
}

func (eq *EventQueue) HandleFlushResults(successPayloads []string, failurePayloads []string, failureWithRetryPayloads []string) {
	eq.stateMutex.Lock()
	defer eq.stateMutex.Unlock()

	var reported int32

	for _, payloadId := range successPayloads {
		if err := eq.reportPayloadSuccess(payloadId); err != nil {
			util.Errorf("failed to mark event payloads as successful: %v", err)
		} else {
			reported++
		}
	}
	for _, payloadId := range failurePayloads {
		if err := eq.reportPayloadFailure(payloadId, false); err != nil {
			util.Errorf("failed to mark event payloads as failed: %v", err)
		} else {
			reported++
		}
	}
	for _, payloadId := range failureWithRetryPayloads {
		if err := eq.reportPayloadFailure(payloadId, true); err != nil {
			util.Errorf("failed to mark event payloads as failed: %v", err)
		} else {
			reported++
		}
	}

	eq.eventsReported.Add(reported)
}

func (eq *EventQueue) Metrics() (int32, int32, int32) {
	return eq.eventsFlushed.Load(), eq.eventsReported.Load(), eq.eventsDropped.Load()
}

func (eq *EventQueue) Close() (err error) {
	eq.done()
	return
}

func (eq *EventQueue) aggQueueLength() int {
	eq.stateMutex.RLock()
	defer eq.stateMutex.RUnlock()
	return len(eq.aggEventQueue)
}

func (eq *EventQueue) UserQueueLength() int {
	eq.stateMutex.RLock()
	defer eq.stateMutex.RUnlock()
	return eq.userEventQueueCount
}

func (eq *EventQueue) updateFailedPayloads() {
	for _, pl := range eq.pendingPayloads {
		if pl.Status == "failed" {
			pl.Status = "sending"
		}
	}
}

func (eq *EventQueue) reportPayloadSuccess(payloadId string) error {
	if _, ok := eq.pendingPayloads[payloadId]; ok {
		delete(eq.pendingPayloads, payloadId)
	} else {
		return fmt.Errorf("Failed to find payload: %s to mark as success", payloadId)
	}
	return nil
}

func (eq *EventQueue) reportPayloadFailure(payloadId string, retryable bool) error {
	if v, ok := eq.pendingPayloads[payloadId]; ok {
		if retryable {
			v.Status = "failed"
		} else {
			delete(eq.pendingPayloads, payloadId)
		}
	} else {
		return fmt.Errorf("Failed to find payload: %s, retryable: %v", payloadId, retryable)
	}
	return nil
}

func (eq *EventQueue) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
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
	eq.stateMutex.Lock()
	defer eq.stateMutex.Unlock()

	// TODO: provide platform data
	popU := event.user.GetPopulatedUser(eq.platformData)
	ccd := GetClientCustomData(eq.sdkKey)
	popU.MergeClientCustomData(ccd)

	bucketedConfig, err := GenerateBucketedConfig(eq.sdkKey, popU, ccd)
	if err != nil {
		return err
	}
	event.event.FeatureVars = bucketedConfig.FeatureVariationMap

	switch event.event.Type_ {
	case api.EventType_AggVariableDefaulted, api.EventType_VariableDefaulted, api.EventType_AggVariableEvaluated, api.EventType_VariableEvaluated:
		break
	default:
		event.event.CustomType = event.event.Type_
		event.event.Type_ = api.EventType_CustomEvent
		event.event.UserId = event.user.UserId
	}

	if _, ok := eq.userEventQueue[popU.UserId]; ok {
		records := eq.userEventQueue[popU.UserId]
		records.Events = append(records.Events, *event.event)
		records.User = popU
		eq.userEventQueue[popU.UserId] = records
	} else {
		record := api.UserEventsBatchRecord{
			User:   popU,
			Events: []api.Event{*event.event},
		}
		eq.userEventQueue[popU.UserId] = record
	}
	eq.userEventQueueCount++
	return nil
}

func (eq *EventQueue) processAggregateEvent(event aggEventData) (err error) {
	defer func() {
		if r := recover(); r != nil {
			util.Warnf("recovered from panic in processAggregateEvent: %v", r)
			if errVal, ok := r.(error); ok {
				err = errVal
			}
		}
	}()

	eq.stateMutex.Lock()
	defer eq.stateMutex.Unlock()
	eType := event.eventType
	eTarget := event.variableKey

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

	if eType == api.EventType_AggVariableEvaluated {
		variationAggMap := make(VariationAggMap)
		if existingVariationAggMap, ok := featureVariationAggregationMap[event.featureId]; ok {
			variationAggMap = existingVariationAggMap
		}
		variationAggMap[event.variationId]++
		featureVariationAggregationMap[event.featureId] = variationAggMap
	} else {
		defaultReasonAggMap := make(VariationAggMap)
		if existingVariationAggMap, ok := featureVariationAggregationMap["defaulted"]; ok {
			defaultReasonAggMap = existingVariationAggMap
		}
		defaultReasonAggMap[event.defaultReason]++
		featureVariationAggregationMap["defaulted"] = defaultReasonAggMap
	}
	variableFeatureVariationAggregationMap[eTarget] = featureVariationAggregationMap
	eq.aggEventQueue[eType] = variableFeatureVariationAggregationMap
	return nil
}
