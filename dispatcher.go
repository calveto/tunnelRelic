package tunnelRelic

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
