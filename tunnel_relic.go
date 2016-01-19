package tunnelRelic

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// A buffered channel that we can send work request on.
var JobQueue chan Job

type Tunnel struct {
	SendInterval    int
	SendBuffer      int
	InsightsAPI     string
	InsightsAccount string
	InsightsEvent   string
	InsightsURL     string
	Silent          bool
	MaxQueue        int
	MaxWorkers      int
}

func NewTunnel(account string, apiKey string, eventName string, send int, sendBuff int) *Tunnel {

	maxWorker, err := strconv.Atoi(os.Getenv("MAX_WORKERS"))
	if err != nil {
		maxWorker = 1
	}
	maxQueue, err := strconv.Atoi(os.Getenv("MAX_QUEUE"))
	if err != nil {
		maxQueue = 100
	}

	url := strings.Join([]string{"https://nsights-collector.newrelic.com/v1/accounts/", account, "/events"}, "")
	relic := &Tunnel{
		SendInterval:    send,
		SendBuffer:      sendBuff,
		InsightsAPI:     apiKey,
		InsightsAccount: account,
		InsightsURL:     url,
		Silent:          false,
		InsightsEvent:   eventName,
		MaxWorkers:      maxWorker,
		MaxQueue:        maxQueue,
	}

	JobQueue = make(chan Job, relic.MaxQueue)
	dispatcher := relic.NewDispatcher(relic.MaxWorkers)
	dispatcher.Run()

	return relic

}

func NewTransaction() map[string]interface{} {
	newRelicTransaction := make(map[string]interface{})

	// Add common attributes for all events
	if hostname, err := os.Hostname(); err == nil {
		newRelicTransaction["host"] = hostname
	} else {
		newRelicTransaction["host"] = "default"
	}

	return newRelicTransaction
}

func (relic *Tunnel) RegisterEvent(event map[string]interface{}) {

	// Create a Job
	fmt.Println("Greating job")
	work := Job{Event: event, EventType: relic.InsightsEvent}

	// Push the work on the queue
	JobQueue <- work
}
