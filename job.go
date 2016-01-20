package insightsRelic

import (
	"encoding/json"
)

type Job struct {
	Event     map[string]interface{}
	EventType string
	Flush     bool
}

func (j Job) String() string {
	j.Event["eventType"] = j.EventType
	eventJson, err := json.Marshal(j.Event)
	if err != nil {
		Log.Errorf("json Marshaling error", err)
	}
	return string(eventJson[:])
}
