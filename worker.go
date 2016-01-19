package tunnelRelic

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

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
	fmt.Println("Sending Batch")
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
