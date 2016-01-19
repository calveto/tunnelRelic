package tunnelRelic

import (
	"encoding/json"
	"fmt"
)

type Job struct {
	Event     map[string]interface{}
	EventType string
}

func (j Job) String() string {
	j.Event["eventType"] = j.EventType
	eventJson, err := json.Marshal(j.Event)
	if err != nil {
		fmt.Println("tunnelRelic: json Marshaling error", err)
	}
	return string(eventJson[:])
}
