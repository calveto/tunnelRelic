package insightsRelic

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// A buffered channel that we can send work request on.
var JobQueue chan Job
var Log Logger

type Insights struct {
	SendInterval    int
	SendBuffer      int
	InsightsAPI     string
	InsightsAccount string
	InsightsEvent   string
	InsightsURL     string
	MaxQueue        int
	MaxWorkers      int
	StripParams     []string
	Log             Logger
}

func NewInsights(account string, apiKey string) *Insights {

	maxWorker, err := strconv.Atoi(os.Getenv("MAX_WORKERS"))
	if err != nil {
		maxWorker = 1
	}
	maxQueue, err := strconv.Atoi(os.Getenv("MAX_QUEUE"))
	if err != nil {
		maxQueue = 100
	}

	url := strings.Join([]string{"https://insights-collector.newrelic.com/v1/accounts/", account, "/events"}, "")
	i := &Insights{
		SendInterval:    10,
		SendBuffer:      5,
		InsightsAPI:     apiKey,
		InsightsAccount: account,
		InsightsURL:     url,
		InsightsEvent:   "Transaction",
		MaxWorkers:      maxWorker,
		MaxQueue:        maxQueue,
		Log:             NewStderrLogger(),
	}

	JobQueue = make(chan Job, i.MaxQueue)
	i.Log.EnableDebug()
	Log = i.Log
	dispatcher := i.NewDispatcher(i.MaxWorkers)
	Log.Info("Starting Dispatcher")
	dispatcher.Run()
	return i
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

func (i *Insights) RegisterEvent(event map[string]interface{}) {

	for _, key := range i.StripParams {
		delete(event, key)
	}
	event["timestamp"] = time.Now().Unix()

	// Create a Job
	work := Job{Event: event, EventType: i.InsightsEvent}
	Log.Debug("Adding Job: ", work.String())

	// Push the work on the queue
	JobQueue <- work
}
