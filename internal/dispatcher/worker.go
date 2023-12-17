package dispatcher

import (
	"fmt"
	"time"
)

type Worker struct {
	ID         int
	Work       chan Work
	WorkerPool chan chan Work
	Quit       chan bool
}

func NewWorker(id int, workerPool chan chan Work) Worker {
	return Worker{
		ID:         id,
		Work:       make(chan Work),
		WorkerPool: workerPool,
		Quit:       make(chan bool),
	}
}

func (w Worker) Start() {
	go func() {
		for {
			w.WorkerPool <- w.Work

			select {
			case work := <-w.Work:
				// do work
				fmt.Printf("worker%d: started %s\n", w.ID, work.Operation)
				time.Sleep(500 * time.Millisecond)
				fmt.Printf("worker%d: finished %s\n", w.ID, work.Operation)
			case <-w.Quit:
				fmt.Printf("worker%d: quitting\n", w.ID)
				return
			}
		}
	}()
}
