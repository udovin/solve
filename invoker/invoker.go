package invoker

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type Invoker struct {
	core  *core.Core
	mutex sync.Mutex
}

var errEmptyQueue = errors.New("empty queue")

func New(c *core.Core) *Invoker {
	return &Invoker{core: c}
}

func (s *Invoker) Start() {
	threads := s.core.Config.Invoker.Threads
	if threads <= 0 {
		threads = 1
	}
	for i := 0; i < threads; i++ {
		s.core.StartTask(s.runDaemon)
	}
}

func (s *Invoker) runDaemon(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var action models.Action
			if err := s.core.WithTx(ctx, func(tx *sql.Tx) error {
				var err error
				action, err = s.core.Actions.PopQueuedTx(tx)
				return err
			}); err != nil && err != sql.ErrNoRows {
				log.Println("Error:", err)
				continue
			}
			s.onAction(action)
		}
	}
}

func (s *Invoker) onAction(action models.Action) {}
