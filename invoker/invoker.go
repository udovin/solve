package invoker

import (
	"database/sql"
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
	return &Invoker{app: app}
}

func (s *Invoker) Start() {
	threads := s.app.Config.Invoker.Threads
	if threads <= 0 {
		threads = 1
	}
	s.closer = make(chan struct{})
	s.waiter.Add(threads)
	for i := 0; i < threads; i++ {
		go s.loop()
	}
}

// Stop stops the invoker.
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
			var action models.Action
			if err := s.app.WithTx(func(tx *sql.Tx) error {
				var err error
				action, err = s.app.Actions.PopQueuedTx(tx)
				return err
			}); err != nil {
				log.Println("Error:", err)
				continue
			}
			s.onAction(action)
		}
	}
}

func (s *Invoker) onAction(action models.Action) {

}
