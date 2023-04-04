package native_bucketing

import (
	"context"
	"fmt"

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

var aggEventQueueRaw = make(map[string]chan aggEventData)
var userEventQueueRaw = make(map[string]chan userEventData)

// sdkKey: map[aggregateByVariation:map[variableVariationMap:map[event:map[target:count]]]]
var aggEventQueue = make(map[string]AggregateEventQueue)
var userEventQueue = make(map[string]map[string]api.UserEventsBatchRecord)
var eventQueueOptions = make(map[string]*api.EventQueueOptions)

func InitEventQueue(sdkKey string, options *api.EventQueueOptions) error {
	if sdkKey == "" {
		return fmt.Errorf("sdk key is required")
	}
	if _, ok := userEventQueueRaw[sdkKey]; ok {
		return fmt.Errorf("event queue already initialized for sdk key: %s", sdkKey)
	}

	userEventQueueRaw[sdkKey] = make(chan userEventData, options.MaxEventQueueSize)
	aggEventQueueRaw[sdkKey] = make(chan aggEventData, options.MaxEventQueueSize)
	eventQueueOptions[sdkKey] = options
	userEventQueue[sdkKey] = make(map[string]api.UserEventsBatchRecord)
	aggEventQueue[sdkKey] = make(AggregateEventQueue)

	go processEvents(sdkKey, context.Background())
	return nil
}

func mergeAggEventQueueKeys(config *configBody, sdkKey string) {
	for _, target := range []string{api.EventType_AggVariableEvaluated, api.EventType_AggVariableDefaulted, api.EventType_VariableEvaluated, api.EventType_VariableDefaulted} {
		if _, ok := aggEventQueue[sdkKey][target]; !ok {
			aggEventQueue[sdkKey][target] = make(VariableAggMap, len(config.Variables))
		}
		for _, variable := range config.Variables {
			if _, ok := aggEventQueue[sdkKey][target][variable.Key]; !ok {
				aggEventQueue[sdkKey][target][variable.Key] = make(FeatureAggMap, len(config.Features))
			}
			for _, feature := range config.Features {
				if _, ok := aggEventQueue[sdkKey][target][feature.Key]; !ok {
					aggEventQueue[sdkKey][target][variable.Key][feature.Key] = make(VariationAggMap, len(feature.Variations))
				}
				for _, variation := range feature.Variations {
					if _, ok := aggEventQueue[sdkKey][target][feature.Key][variation.Key]; !ok {
						aggEventQueue[sdkKey][target][variable.Key][feature.Key][variation.Key] = 0
					}
				}
			}
		}
	}
}

// QueueAggregateEvent queues an aggregate event to be sent to the server - but offloads actual computing of the event itself
// to a different goroutine.
func QueueAggregateEvent(sdkKey string, event *api.DVCEvent, variableVariationMap map[string]FeatureVariation, aggregateByVariation bool) error {
	if opt, ok := eventQueueOptions[sdkKey]; ok && opt.IsEventLoggingDisabled(event) {
		return nil
	}
	if sdkKey == "" {
		return fmt.Errorf("sdk key is required")
	}
	if event.Target == "" {
		return fmt.Errorf("target is required for aggregate events")
	}
	if _, ok := aggEventQueueRaw[sdkKey]; !ok {
		return fmt.Errorf("event queue not initialized for sdk key: %s", sdkKey)
	}

	aggEventQueueRaw[sdkKey] <- aggEventData{
		event:                event,
		variableVariationMap: variableVariationMap,
		aggregateByVariation: aggregateByVariation,
	}
	return nil
}

func QueueEvent(sdkKey string, user DVCUser, event api.DVCEvent) error {
	if sdkKey == "" {
		return fmt.Errorf("sdk key is required")
	}
	if _, ok := userEventQueueRaw[sdkKey]; !ok {
		return fmt.Errorf("event queue not initialized for sdk key: %s", sdkKey)
	}

	userEventQueueRaw[sdkKey] <- userEventData{
		event: &event,
		user:  &user,
	}

	return nil
}

func Close(sdkKey string) {
	close(userEventQueueRaw[sdkKey])
	close(aggEventQueueRaw[sdkKey])
}

func processEvents(sdkKey string, ctx context.Context) {
	for {
		select {
		case _ = <-ctx.Done():
			return
		default:
		}
		select {
		case userEvent := <-userEventQueueRaw[sdkKey]:
			processUserEvent(sdkKey, userEvent)
		case aggEvent := <-aggEventQueueRaw[sdkKey]:
			processAggregateEvent(sdkKey, aggEvent)
		default:
		}
	}
}

func processUserEvent(sdkKey string, event userEventData) error {
	// TODO: provide platform data
	popU := event.user.GetPopulatedUser(nil)
	ccd := GetClientCustomData(sdkKey)
	popU.MergeClientCustomData(ccd)

	bucketedConfig, err := GenerateBucketedConfig(sdkKey, popU, ccd)
	if err != nil {
		// TODO: Log
		return err
	}
	event.event.FeatureVars = bucketedConfig.FeatureVariationMap
	if _, ok := userEventQueue[sdkKey][popU.UserId]; ok {
		records := userEventQueue[sdkKey][popU.UserId]
		records.Events = append(records.Events, *event.event)
		records.User = popU
		userEventQueue[sdkKey][popU.UserId] = records
	} else {
		record := api.UserEventsBatchRecord{
			User:   popU,
			Events: []api.DVCEvent{*event.event},
		}
		userEventQueue[sdkKey][popU.UserId] = record
	}
	return nil
}

func processAggregateEvent(sdkKey string, event aggEventData) error {
	eType := event.event.Type_
	eTarget := event.event.Target

	variableFeatureVariationAggregationMap := make(VariableAggMap)
	if v, ok := aggEventQueue[sdkKey][eType]; ok {
		variableFeatureVariationAggregationMap = v
	} else {
		aggEventQueue[sdkKey][eType] = variableFeatureVariationAggregationMap
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
		} else {
			featureVariationAggregationMap[featureVar.Feature] = variationAggMap
		}

		if _, ok := variationAggMap[featureVar.Variation]; ok {
			variationAggMap[featureVar.Variation] += 1
		} else {
			variationAggMap[featureVar.Variation] = 1
			// increment event queue count
		}
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
