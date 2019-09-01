package worker

import (
	"sync"
	"time"

	"github.com/udovin/solve/core"
)

type Worker struct {
	app    *core.App
	closer chan struct{}
	waiter sync.WaitGroup
}

func New(app *core.App) *Worker {
	return &Worker{
		app: app,
	}
}

func (w *Worker) Start() {
	w.waiter.Add(1)
	w.closer = make(chan struct{})
	go w.loop()
}

func (w *Worker) Stop() {
	close(w.closer)
	w.waiter.Wait()
}

func (w *Worker) loop() {
	defer w.waiter.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-w.closer:
			return
		case <-ticker.C:

		}
	}
}
