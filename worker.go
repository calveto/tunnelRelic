package tunnelRelic

import (
	"bytes"
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
	Log.Debug("Starting Worker")
	go func() {
		jobs := []Job{}

		for {

			// register the current worker into the worker queue
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:

				// we have received a work request.
				jobs = append(jobs, job)

				if len(jobs) > w.Config.SendBuffer || job.Flush {
					go w.SendBatch(jobs)
					jobs = []Job{}
				}
			case <-w.quit:
				return
			}
		}
	}()
}

func (w Worker) SendBatch(jobs []Job) {

	var events []string
	for _, job := range jobs {
		if !job.Flush {
			events = append(events, job.String())
		}
	}

	if len(events) == 0 {
		return
	}

	requestStr := "[" + strings.Join(events, ",") + "]"

	var eventJson = []byte(requestStr)
	Log.Debugf("EventJson: \n %s", eventJson)

	req, err := http.NewRequest("POST", w.Config.InsightsURL, bytes.NewBuffer(eventJson))
	if err != nil {
		Log.Error(err.Error())
		return
	}
	req.Header.Set("X-Insert-Key", w.Config.InsightsAPI)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		Log.Error(err.Error())
		return
	}

	if resp.StatusCode >= 500 {
		// Re-queue failed post
		Log.Warn("Re-queueing jobs")
		for _, job := range jobs {
			JobQueue <- job
		}
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.Error("HTTP Status Code: ", resp.Status)
		Log.Error("Error response from New Relic: ", string(body))
		return
	}
	if w.Config.Silent != true {
		Log.Info("Sending queued request to New Relic. Response: ", string(body))
	}
}

// Stop the worker
func (w Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}
