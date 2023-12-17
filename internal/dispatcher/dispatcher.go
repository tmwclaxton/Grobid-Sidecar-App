package dispatcher

type dispatcher struct {
	workers    []Worker
	workerPool chan chan Work
	work       chan Work
	quit       chan bool
}

// NewDispatcher creates a new Dispatcher object
func NewDispatcher(workerCount int) *dispatcher {
	return &dispatcher{
		workers:    make([]Worker, workerCount),
		workerPool: make(chan chan Work, workerCount),
		work:       make(chan Work),
		quit:       make(chan bool),
	}
}

// Start starts the dispatcher
func (d *dispatcher) Start() {
	for i := 0; i < len(d.workers); i++ {
		d.workers[i] = NewWorker(i+1, d.workerPool)
		d.workers[i].Start()
	}
}

// Stop stops the dispatcher
func (d *dispatcher) Stop() {
	go func() {
		d.quit <- true
	}()
}

// Dispatch dispatches work to the workers
func (d *dispatcher) Dispatch(work Work) {
	d.work <- work
}
