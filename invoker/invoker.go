package invoker

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

// Invoker represents manager for asynchronous actions (invocations).
type Invoker struct {
	core  *core.Core
	mutex sync.Mutex
}

// New creates a new instance of Invoker.
func New(c *core.Core) *Invoker {
	return &Invoker{core: c}
}

// Start starts invoker daemons.
//
// This function will spawn config.Invoker.Threads amount of goroutines.
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
			if err := s.onAction(action); err != nil {
				action.Status = models.Failed
			} else {
				action.Status = models.Succeeded
			}
			if err := s.core.WithTx(ctx, func(tx *sql.Tx) error {
				return s.core.Actions.UpdateTx(tx, action)
			}); err != nil && err != sql.ErrNoRows {
				log.Println("Error:", err)
			}
		}
	}
}

func (s *Invoker) onAction(action models.Action) error {
	s.core.Logger().Debug("Received new action %d", action.ID)
	switch action.Type {
	case models.JudgeSolution:
		return s.onJudgeSolution(action)
	default:
		s.core.Logger().Errorf("Unknown action: %v", action.Type)
		return fmt.Errorf("unknown action")
	}
}

func (s *Invoker) onJudgeSolution(action models.Action) error {
	return fmt.Errorf("not implemented")
}
