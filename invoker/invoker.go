package invoker

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

// Invoker represents manager for asynchronous actions (invocations).
type Invoker struct {
	core *core.Core
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
		default:
			if ok := s.runDaemonTick(ctx); !ok {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}
	}
}

func (s *Invoker) runDaemonTick(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	var task models.Task
	if err := s.core.WithTx(ctx, func(tx *sql.Tx) error {
		var err error
		task, err = s.core.Tasks.PopQueuedTx(tx)
		return err
	}); err != nil {
		if err != sql.ErrNoRows {
			s.core.Logger().Error("Unable to pop task", zap.Error(err))
		}
		return false
	}
	defer func() {
		if r := recover(); r != nil {
			task.Status = models.Failed
			s.core.Logger().Error("Task panic", zap.Any("panic", r))
		}
		ctx, cancel := context.WithDeadline(context.Background(), time.Unix(task.ExpireTime, 0))
		defer cancel()
		if err := s.core.WithTx(ctx, func(tx *sql.Tx) error {
			return s.core.Tasks.UpdateTx(tx, task)
		}); err != nil {
			s.core.Logger().Error("Unable to update task", zap.Error(err))
		}
	}()
	var waiter sync.WaitGroup
	defer waiter.Wait()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	waiter.Add(1)
	go func() {
		defer waiter.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				select {
				case <-ctx.Done():
					return
				default:
				}
				if time.Now().After(time.Unix(task.ExpireTime, 0)) {
					s.core.Logger().Error(
						"Task expired",
						zap.Int64("task_id", task.ID),
					)
					return
				}
				clone := task
				if err := s.core.WithTx(ctx, func(tx *sql.Tx) error {
					clone.ExpireTime = time.Now().Add(5 * time.Second).Unix()
					return s.core.Tasks.UpdateTx(tx, clone)
				}); err != nil {
					s.core.Logger().Error(
						"Unable to ping task",
						zap.Int64("task_id", task.ID),
						zap.Error(err),
					)
				} else {
					task.ExpireTime = clone.ExpireTime
				}
			}
		}
	}()
	err := s.onTask(ctx, task)
	cancel()
	waiter.Wait()
	if err != nil {
		task.Status = models.Failed
	} else {
		task.Status = models.Succeeded
	}
	return true
}

func (s *Invoker) onTask(ctx context.Context, task models.Task) error {
	s.core.Logger().Debug(
		"Received new task",
		zap.Int64("task_id", task.ID),
	)
	switch task.Kind {
	case models.JudgeSolution:
		return s.onJudgeSolution(ctx, task)
	default:
		s.core.Logger().Error(
			"Unknown task",
			zap.Int64("task_id", task.ID),
			zap.Int("task_kind", int(task.Kind)),
		)
		return fmt.Errorf("unknown task")
	}
}

func (s *Invoker) onJudgeSolution(ctx context.Context, task models.Task) error {
	return fmt.Errorf("not implemented")
}
