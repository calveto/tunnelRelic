package tunnelRelic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	//"os"
	"strings"
	"time"
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

// A buffered channel that we can send work request on.
var JobQueue = make(chan Job, 100)

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
					fmt.Println("---------------------- Doing some Work!! -------------------------")

					// pop one job at a time off the slice
					var jobBatch []Job
					for i := 0; i < w.Config.SendBuffer; i++ {
						job := Job{}
						job, jobs = jobs[len(jobs)-1], jobs[:len(jobs)-1]
						jobBatch = append(jobBatch, job)
					}

					// Check this out to make sure there is not race condition
					// Run code with race detector
					go w.SendBatch(jobBatch)
					jobBatch = []Job{}
				} else {
					fmt.Printf("\nBuffering: We only have %d Jobs in the slice ---->\n", len(jobs))
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
	fmt.Printf("\nSending Events to NR --------------------------->")
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
	// Need to check status codes and re-queue when appropriate.

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
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

	url := strings.Join([]string{"https://nsights-collector.newrelic.com/v1/accounts/", account, "/events"}, "")
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

	//	go relic.MaintainQueue()
	dispatcher := relic.NewDispatcher(1)
	dispatcher.Run()

	return relic

}

func NewTransaction() map[string]interface{} {
	newRelicTransaction := make(map[string]interface{})
	// Add common attributes for all events
	/*
		if hostname, err := os.Hostname(); err == nil {
			newRelicTransaction["host"] = hostname
		} else {
			newRelicTransaction["host"] = "default"
		}
	*/

	return newRelicTransaction
}

func (relic *Tunnel) RegisterEvent(event map[string]interface{}) {
	// Create a Job
	work := Job{Event: event, EventType: relic.InsightsEvent}

	// Push the work on the queue
	JobQueue <- work
}
func (relic *Tunnel) MaintainQueue() {

	for true {
		time.Sleep(time.Second * time.Duration(int64(relic.SendInterval)))
		//go relic.EmptyQueue()
	}

}

/*
func (relic *Tunnel) EmptyQueue() {

	if len(relic.SendQueue) < 1 {
		return
	}
	if relic.Silent != true {
		fmt.Println("tunnelRelic: Gophers will now proceed to deliver queued events to New Relic.")
	}

	requestStr := "[" + strings.Join(relic.SendQueue, ",") + "]"

	var eventJson = []byte(requestStr)
	req, err := http.NewRequest("POST", relic.InsightsURL, bytes.NewBuffer(eventJson))
	if err != nil {
		relic.SendQueue = nil
		return
	}
	req.Header.Set("X-Insert-Key", relic.InsightsAPI)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	relic.SendQueue = nil

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if relic.Silent != true {
		fmt.Println("tunnelRelic: Sending queued request to New Relic. Response: ", string(body))
	}

}
*/
