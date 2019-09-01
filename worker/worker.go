package worker

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type Worker struct {
	app    *core.App
	closer chan struct{}
	waiter sync.WaitGroup
}

var errEmptyQueue = errors.New("empty queue")

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
			report, err := w.popQueuedReport()
			if err != nil {
				if err != errEmptyQueue {
					log.Println("Error:", err)
				}
				continue
			}
			log.Println(report)
		}
	}
}

func (w *Worker) popQueuedReport() (report models.Report, err error) {
	tx, err := w.app.Reports.Manager.Begin()
	if err != nil {
		return
	}
	if err = w.app.Reports.Manager.SyncTx(tx); err != nil {
		return
	}
	queuedIDs := w.app.Reports.GetQueuedIDs()
	if len(queuedIDs) == 0 {
		if err := tx.Rollback(); err != nil {
			log.Println("Error:", err)
		}
		err = errEmptyQueue
		return
	}
	report, ok := w.app.Reports.Get(queuedIDs[0])
	if !ok {
		err = errEmptyQueue
		return
	}
	report.Verdict = -1
	if err = w.app.Reports.UpdateTx(tx, &report); err != nil {
		if err := tx.Rollback(); err != nil {
			log.Println("Error:", err)
		}
		return
	}
	err = tx.Commit()
	return
}
