package invoker

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type Invoker struct {
	app    *core.Core
	closer chan struct{}
	waiter sync.WaitGroup
	mutex  sync.Mutex
}

var errEmptyQueue = errors.New("empty queue")

func New(app *core.Core) *Invoker {
	return &Invoker{
		app: app,
	}
}

func (s *Invoker) Start() {
	threads := s.app.Config.Invoker.Threads
	if threads <= 0 {
		threads = 1
	}
	s.closer = make(chan struct{})
	for i := 0; i < threads; i++ {
		s.waiter.Add(1)
		go s.loop()
	}
}

func (s *Invoker) Stop() {
	close(s.closer)
	s.waiter.Wait()
}

func (s *Invoker) loop() {
	defer s.waiter.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.closer:
			return
		case <-ticker.C:
			report, err := s.popQueuedReport()
			if err != nil {
				if err != errEmptyQueue {
					log.Println("Error:", err)
				}
				continue
			}
			if err := s.app.Solutions.Manager.Sync(); err != nil {
				log.Println("Error:", err)
			}
			if err := s.app.Compilers.Manager.Sync(); err != nil {
				log.Println("Error:", err)
			}
			solution, err := s.app.Solutions.Get(report.SolutionID)
			if err != nil {
				log.Printf(
					"Unable to find solution for report = %d",
					report.SolutionID,
				)
				continue
			}
			req := context{
				Solution: &solution,
				Report:   &report,
			}
			if err := s.processSolution(&req); err != nil {
				log.Println("Error:", err)
			}
		}
	}
}

func (s *Invoker) popQueuedReport() (report models.Report, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	tx, err := s.app.Reports.Manager.Begin()
	if err != nil {
		return
	}
	if err = s.app.Reports.Manager.SyncTx(tx); err != nil {
		return
	}
	queuedIDs := s.app.Reports.GetQueuedIDs()
	if len(queuedIDs) == 0 {
		if err := tx.Rollback(); err != nil {
			log.Println("Error:", err)
		}
		err = errEmptyQueue
		return
	}
	report, err = s.app.Reports.Get(queuedIDs[0])
	if err != nil {
		err = errEmptyQueue
		return
	}
	report.Verdict = -1
	if err = s.app.Reports.UpdateTx(tx, &report); err != nil {
		if err := tx.Rollback(); err != nil {
			log.Println("Error:", err)
		}
		return
	}
	err = tx.Commit()
	return
}
