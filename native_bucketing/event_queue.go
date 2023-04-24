package native_bucketing

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/devcyclehq/go-server-sdk/v2/api"
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

type VariationAggMap = map[string]int64
type FeatureAggMap = map[string]VariationAggMap
type VariableAggMap = map[string]FeatureAggMap
type AggregateEventQueue = map[string]VariableAggMap

type EventQueue struct {
	sdkKey            string
	options           *api.EventQueueOptions
	aggEventQueueRaw  chan aggEventData
	userEventQueueRaw chan userEventData
	userEventQueue    map[string]api.UserEventsBatchRecord
	aggEventQueue     map[string]map[string]map[string]map[string]int64
	aggEventMutex     *sync.RWMutex
	eventsFlushed     atomic.Int32
	eventsReported    atomic.Int32
}

func InitEventQueue(sdkKey string, options *api.EventQueueOptions) (*EventQueue, error) {
	if sdkKey == "" {
		return nil, fmt.Errorf("sdk key is required")
	}

	eq := &EventQueue{
		sdkKey:            sdkKey,
		options:           options,
		aggEventQueueRaw:  make(chan aggEventData, options.MaxEventQueueSize),
		userEventQueueRaw: make(chan userEventData, options.MaxEventQueueSize),
		userEventQueue:    make(map[string]api.UserEventsBatchRecord),
		aggEventQueue:     make(AggregateEventQueue),
		aggEventMutex:     &sync.RWMutex{},
	}
	//go eq.processEvents(context.Background())
	return eq, nil
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

	eq.aggEventQueueRaw <- aggEventData{
		event:                &event,
		variableVariationMap: config.VariableVariationMap,
		aggregateByVariation: event.Type_ == api.EventType_AggVariableEvaluated,
	}
	return nil
}

func (eq *EventQueue) QueueEvent(user DVCUser, event api.DVCEvent) error {
	eq.userEventQueueRaw <- userEventData{
		event: &event,
		user:  &user,
	}

	return nil
}

func (eq *EventQueue) FlushEvents() (err error) {
	return nil
}

func (eq *EventQueue) Metrics() (int32, int32) {
	return eq.eventsFlushed.Load(), eq.eventsReported.Load()
}

func (eq *EventQueue) Close() (err error) {
	err = eq.FlushEvents()
	close(eq.userEventQueueRaw)
	close(eq.aggEventQueueRaw)
	return
}

func (eq *EventQueue) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		select {
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
		default:
		}
	}
}

func (eq *EventQueue) processUserEvent(event userEventData) error {
	// TODO: provide platform data
	popU := event.user.GetPopulatedUser(nil)
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

func (eq *EventQueue) processAggregateEvent(event aggEventData) error {
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
			if variationAggMap, ok2 := feature["value"]; ok2 {
				variationAggMap += 1
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
