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

var ErrQueueFull = fmt.Errorf("max queue size reached")

type aggEventData struct {
	eventType   string
	variableKey string
	featureId   string
	variationId string
	evalDetails string
	evalReason  EvaluationReason
}

type userEventData struct {
	event *api.Event
	user  *api.User
}

// Structure of the aggregation maps
// map event type -> event target
// map event target -> feature id
// map feature id -> variation id
// map variation -> eval reason count
// For Evaluation Events:
// ["aggVariableEvaluated"]["somevariablekey"]["feature_id"]["variation_id"]["eval reason"] = 1
// For Defaulted Events:
// ["aggVariableDefaulted"]["somevariablekey"]["DEFAULT"]["DEFAULT_REASON"] = 1

type EvalReasonAggMap map[EvaluationReason]int64
type VariationAggMap map[string]EvalReasonAggMap
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

func (agg *AggregateEventQueue) BuildBatchRecords(platformData *api.PlatformData, clientUUID, configEtag, rayId, lastModified string) api.UserEventsBatchRecord {
	var aggregateEvents []api.Event
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "aggregate"
	}
	userId := fmt.Sprintf("%s@%s", clientUUID, hostname)
	emptyFeatureVars := make(map[string]string)

	// type is either aggVariableEvaluated or aggVariableDefaulted
	for _type, variableAggMap := range *agg {
		for variableKey, featureAggMap := range variableAggMap {
			// feature is feature id for evaluation events, or the string "defaulted" for default events
			for feature, _variationAggMap := range featureAggMap {
				// variation is variation id for evaluation events, or the "default reason" for default events
				for variation, evalReason := range _variationAggMap {
					event := api.Event{
						Type_:       _type,
						Target:      variableKey,
						UserId:      userId,
						FeatureVars: emptyFeatureVars,
						ClientDate:  time.Now(),
					}
					metaData := make(map[string]interface{})
					evalMetadata := make(map[string]int64)
					if _type == api.EventType_AggVariableDefaulted {
						metaData["evalDetails"] = variation
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
					if lastModified != "" {
						metaData["configLastModified"] = lastModified
					}
					if _type == api.EventType_AggVariableEvaluated || _type == api.EventType_AggVariableDefaulted {
						for reason, count := range evalReason {
							if count == 0 {
								continue
							}
							evalMetadata[string(reason)] = count
							event.Value += float64(count)
						}
						metaData["eval"] = evalMetadata
					}
					event.MetaData = metaData
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
	queueAccess         *sync.RWMutex
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
		queueAccess:       &sync.RWMutex{},
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
	eq.queueAccess.Lock()
	defer eq.queueAccess.Unlock()

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
						eq.aggEventQueue[target][variable.Key][feature.Key][variation.Key] = make(EvalReasonAggMap)
					}
					for _, reason := range allEvalReasons {
						if _, ok := eq.aggEventQueue[target][variable.Key][feature.Key][variation.Key][reason]; !ok {
							eq.aggEventQueue[target][variable.Key][feature.Key][variation.Key][reason] = 0
						}
					}
				}
			}
		}
	}
}

func (eq *EventQueue) queueAggregateEventInternal(variableKey, featureId, variationId, eventType string, evalReason EvaluationReason, evalDetails string) error {
	if eq.options != nil && eq.options.IsEventLoggingDisabled(eventType) {
		return nil
	}

	if variableKey == "" {
		return fmt.Errorf("a variable key is required for aggregate events")
	}

	select {
	case eq.aggEventQueueRaw <- aggEventData{
		eventType:   eventType,
		variableKey: variableKey,
		featureId:   featureId,
		variationId: variationId,
		evalReason:  evalReason,
		evalDetails: evalDetails,
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

func (eq *EventQueue) QueueVariableEvaluatedEvent(variableKey, featureId, variationId string, evalReason EvaluationReason) error {
	if eq.options.DisableAutomaticEventLogging {
		return nil
	}

	return eq.queueAggregateEventInternal(variableKey, featureId, variationId, api.EventType_AggVariableEvaluated, evalReason, "")
}

func (eq *EventQueue) QueueVariableDefaultedEvent(variableKey string, defaultReason DefaultReason) error {
	if eq.options.DisableAutomaticEventLogging {
		return nil
	}

	return eq.queueAggregateEventInternal(variableKey, "", "", api.EventType_AggVariableDefaulted, EvaluationReasonDefault, string(defaultReason))
}

func (eq *EventQueue) FlushEventQueue(clientUUID, configEtag, rayId, lastModified string) (map[string]api.FlushPayload, error) {
	eq.queueAccess.Lock()
	defer eq.queueAccess.Unlock()

	var records []api.UserEventsBatchRecord

	records = append(records, eq.aggEventQueue.BuildBatchRecords(eq.platformData, clientUUID, configEtag, rayId, lastModified))
	records = append(records, eq.userEventQueue.BuildBatchRecords()...)
	eq.aggEventQueue = make(AggregateEventQueue)
	eq.userEventQueue = make(UserEventQueue)

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
	eq.userEventQueueCount = 0
	eq.updateFailedPayloads()

	eq.eventsFlushed.Add(int32(len(eq.pendingPayloads)))

	return eq.pendingPayloads, nil
}

func (eq *EventQueue) HandleFlushResults(successPayloads []string, failurePayloads []string, failureWithRetryPayloads []string) {
	eq.queueAccess.Lock()
	defer eq.queueAccess.Unlock()

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

// Metrics returns the number of events flushed, reported, and dropped
func (eq *EventQueue) Metrics() (int32, int32, int32) {
	return eq.eventsFlushed.Load(), eq.eventsReported.Load(), eq.eventsDropped.Load()
}

func (eq *EventQueue) Close() (err error) {
	eq.done()
	return
}

func (eq *EventQueue) aggQueueLength() int {
	eq.queueAccess.RLock()
	defer eq.queueAccess.RUnlock()
	return len(eq.aggEventQueue)
}

func (eq *EventQueue) UserQueueLength() int {
	eq.queueAccess.RLock()
	defer eq.queueAccess.RUnlock()
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
		return fmt.Errorf("failed to find payload: %s to mark as success", payloadId)
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
		return fmt.Errorf("failed to find payload: %s, retryable: %v", payloadId, retryable)
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
		case userEvent, ok := <-eq.userEventQueueRaw:
			// if the channel is closed - ok will be false
			if !ok {
				return
			}
			err := eq.processUserEvent(userEvent)
			if err != nil {
				return
			}
		case aggEvent, ok := <-eq.aggEventQueueRaw:
			// if the channel is closed - ok will be false
			if !ok {
				return
			}
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

	// Get user data and custom data without holding the queue lock
	popU := event.user.GetPopulatedUser(eq.platformData)
	ccd := GetClientCustomData(eq.sdkKey)
	popU.MergeClientCustomData(ccd)

	// Generate bucketed config without holding the queue lock to avoid deadlock
	bucketedConfig, err := GenerateBucketedConfig(eq.sdkKey, popU, ccd)
	if err != nil {
		return err
	}
	event.event.FeatureVars = bucketedConfig.FeatureVariationMap

	switch event.event.Type_ {
	case api.EventType_AggVariableDefaulted, api.EventType_VariableDefaulted, api.EventType_AggVariableEvaluated, api.EventType_VariableEvaluated, api.EventType_SDKConfig:
		break
	default:
		event.event.CustomType = event.event.Type_
		event.event.Type_ = api.EventType_CustomEvent
		event.event.UserId = event.user.UserId
	}

	// Only acquire the queue lock for the final queue operations
	eq.queueAccess.Lock()
	defer eq.queueAccess.Unlock()

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

	eq.queueAccess.Lock()
	defer eq.queueAccess.Unlock()
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
		if _, ok := variationAggMap[event.variationId]; !ok {
			variationAggMap[event.variationId] = make(EvalReasonAggMap)
		}
		evalReasons := variationAggMap[event.variationId]
		if _, ok := evalReasons[event.evalReason]; !ok {
			evalReasons[event.evalReason] = 0
		}
		evalReasons[event.evalReason]++
		variationAggMap[event.variationId] = evalReasons
		featureVariationAggregationMap[event.featureId] = variationAggMap
	} else {
		defaultReasonAggMap := make(VariationAggMap)
		if existingVariationAggMap, ok := featureVariationAggregationMap[string(EvaluationReasonDefault)]; ok {
			defaultReasonAggMap = existingVariationAggMap
		}
		// Default events have no variation; only a static default flag to then aggregate by default reason.
		// To make the aggregation mapping consistent later on when re-aggregating - it will result in a double aggregation of `[default][default][reason]` intentionally.
		if _, ok := defaultReasonAggMap[string(EvaluationReasonDefault)]; !ok {
			defaultReasonAggMap[string(EvaluationReasonDefault)] = make(EvalReasonAggMap)
		}
		defaultReasons := defaultReasonAggMap[string(EvaluationReasonDefault)]
		_defaultDetails := EvaluationReason(event.evalDetails)
		if _, ok := defaultReasons[_defaultDetails]; !ok {
			defaultReasons[_defaultDetails] = 0
		}
		defaultReasons[_defaultDetails]++
		defaultReasonAggMap[string(EvaluationReasonDefault)] = defaultReasons
		featureVariationAggregationMap[string(EvaluationReasonDefault)] = defaultReasonAggMap
	}
	variableFeatureVariationAggregationMap[eTarget] = featureVariationAggregationMap
	eq.aggEventQueue[eType] = variableFeatureVariationAggregationMap
	return nil
}
