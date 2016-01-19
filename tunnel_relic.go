package tunnelRelic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// A buffered channel that we can send work request on.
var JobQueue = make(chan Job, 100)

type Tunnel struct {
	SendInterval    int
	SendBuffer      int
	MaxWorkers      int
	InsightsAPI     string
	InsightsAccount string
	InsightsEvent   string
	InsightsURL     string
	Silent          bool
	SendQueue       []string
}

func NewTunnel(account string, apiKey string, eventName string, send int, sendBuff int, maxWorkers int) *Tunnel {

	url := strings.Join([]string{"https://insights-collector.newrelic.com/v1/accounts/", account, "/events"}, "")
	relic := &Tunnel{
		SendInterval:    send,
		SendBuffer:      sendBuff,
		InsightsAPI:     apiKey,
		InsightsAccount: account,
		InsightsURL:     url,
		Silent:          false,
		InsightsEvent:   eventName,
		MaxWorkers:      maxWorkers,
	}

	dispatcher := relic.NewDispatcher(maxWorkers)
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
	work := Job{Event: event, EventType: relic.InsightsEvent}

	// Push the work on the queue
	JobQueue <- work
}

type Dispatcher struct {
	WorkerPool chan chan Job // A pool of workers channels that are registered with the dispatcher
	Config     *Tunnel       // Configuration
}

func (relic *Tunnel) NewDispatcher(numWorkers int) *Dispatcher {
	pool := make(chan chan Job, numWorkers)
	return &Dispatcher{WorkerPool: pool, Config: relic}
}

// Start MaxWorkers number of Workers
func (d *Dispatcher) Run() {
	for i := 0; i < d.Config.MaxWorkers; i++ {
		worker := NewWorker(d.WorkerPool, d.Config)
		worker.Start()
	}
	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-JobQueue:
			// a job request has been received
			go func(job Job) {
				// try to obtain a worker job channel that is available
				// this will block until a worker is idle
				jobChannel := <-d.WorkerPool

				// dispatch the job to the worker job channel
				jobChannel <- job
			}(job)
		}
	}
}

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

// Worker represents the worker that executes the job.
type Worker struct {
	WorkerPool chan chan Job
	JobChannel chan Job
	quit       chan bool
	Config     *Tunnel
}

func NewWorker(workerPool chan chan Job, config *Tunnel) Worker {
	w := Worker{
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool)}

	w.Config = config
	return w
}

// Start method starts the run loop for the worker.
func (w Worker) Start() {
	go func() {
		jobs := []Job{}
		for {
			// register the current worker into the worker queue
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				// we have received a work request.
				jobs = append(jobs, job)

				if len(jobs) > w.Config.SendBuffer {
					// pop off the jobs
					var jobBatch []Job
					for i := 0; i < w.Config.SendBuffer; i++ {
						job := Job{}
						job, jobs = jobs[len(jobs)-1], jobs[:len(jobs)-1]
						jobBatch = append(jobBatch, job)
					}
					go w.SendBatch(jobBatch)
					jobBatch = []Job{}
				}
			case <-w.quit:
				return
			}
		}
	}()
}

func (w Worker) SendBatch(jobs []Job) {
	var events []string
	for x := range jobs {
		events = append(events, jobs[x].String())
	}
	requestStr := "[" + strings.Join(events, ",") + "]"

	var eventJson = []byte(requestStr)
	fmt.Printf("\nEvent Json: \n\n%s\n\n", eventJson)

	req, err := http.NewRequest("POST", w.Config.InsightsURL, bytes.NewBuffer(eventJson))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	req.Header.Set("X-Insert-Key", w.Config.InsightsAPI)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if resp.StatusCode >= 500 {
		// Re-queue failed post
		// Consider adding exponential back off on these retries.
		for _, job := range jobs {
			JobQueue <- job
		}
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("tunnelRelic: HTTP Status Code: ", resp.Status)
		fmt.Println("tunnelRelic: Error response from New Relic: ", string(body))
		return
	}
	if w.Config.Silent != true {
		fmt.Println("tunnelRelic: Sending queued request to New Relic. Response: ", string(body))
	}
}

// Stop the worker
func (w Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}
