package api

type ClientEvent struct {
	EventType ClientEventType `json:"eventType"`
	EventData interface{}     `json:"eventData"`
	Status    string          `json:"status"`
	Error     error           `json:"error"`
}

type ClientEventType string

const (
	ClientEventType_Initialized                ClientEventType = "initialized"
	ClientEventType_Error                      ClientEventType = "error"
	ClientEventType_ConfigUpdated              ClientEventType = "configUpdated"
	ClientEventType_RealtimeUpdates            ClientEventType = "realtimeUpdates"
	ClientEventType_InternalSSEFailure         ClientEventType = "internalSSEFailure"
	ClientEventType_InternalNewConfigAvailable ClientEventType = "internalNewConfigAvailable"
	ClientEventType_InternalSSEConnected       ClientEventType = "internalSSEConnected"
)
