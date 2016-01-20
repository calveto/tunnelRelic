package insightsRelic

import (
	"time"
)

type Dispatcher struct {
	WorkerPool      chan chan Job // A pool of workers channels that are registered with the dispatcher
	Insights        *Insights
	WorkerInstances []Worker
}

func (i *Insights) NewDispatcher(numWorkers int) *Dispatcher {
	pool := make(chan chan Job, numWorkers)
	d := Dispatcher{WorkerPool: pool, Insights: i}
	go d.PeriodicallyFlushWorkers()
	return &d
}

func (d *Dispatcher) PeriodicallyFlushWorkers() {
	for true {
		time.Sleep(time.Second * time.Duration(int64(d.Insights.SendInterval)))
		Log.Debug("Flushing partial batches")
		go func() {
			for _, w := range d.WorkerInstances {
				jobChannel := <-w.WorkerPool
				jobChannel <- Job{Flush: true}
			}
		}()
	}
}

// Start MaxWorkers number of Workers
func (d *Dispatcher) Run() {
	for i := 0; i < d.Insights.MaxWorkers; i++ {
		worker := NewWorker(d.WorkerPool, d.Insights)
		d.WorkerInstances = append(d.WorkerInstances, worker)
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
