// Package invoker represents solve implementation for running asynchronious
// tasks like compiling and judging solutions.
package invoker

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg/logs"
	"github.com/udovin/solve/pkg/safeexec"
)

// Invoker represents manager for asynchronous actions (invocations).
type Invoker struct {
	core      *core.Core
	files     *managers.FileManager
	solutions *managers.SolutionManager
	compilers *compilerManager
	problems  *problemManager
}

// New creates a new instance of Invoker.
func New(core *core.Core) *Invoker {
	s := Invoker{
		core: core,
	}
	if core.Config.Storage != nil {
		s.files = managers.NewFileManager(core)
	}
	if s.files != nil {
		s.solutions = managers.NewSolutionManager(core, s.files)
	}
	return &s
}

// Start starts invoker daemons.
//
// This function will spawn config.Invoker.Workers amount of goroutines.
func (s *Invoker) Start() error {
	safeexecConfig := s.core.Config.Invoker.Safeexec
	safeexec, err := safeexec.NewManager(
		safeexecConfig.Path,
		"/tmp/solve-safeexec",
		"solve-safeexec",
	)
	if err != nil {
		return err
	}
	compilers, err := newCompilerManager(
		s.files, "/tmp/solve-compilers", safeexec, s.core,
	)
	if err != nil {
		return err
	}
	s.compilers = compilers
	problems, err := newProblemManager(
		s.files, "/tmp/solve-problems", compilers,
	)
	if err != nil {
		return err
	}
	s.problems = problems
	workers := s.core.Config.Invoker.Workers
	if workers <= 0 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		name := fmt.Sprintf("invoker-%d", i+1)
		s.core.StartTask(name, s.runDaemon)
	}
	return nil
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
	task, err := popQueuedTask(ctx, s.core.Tasks)
	if err != nil {
		if err != sql.ErrNoRows {
			s.core.Logger().Error("Error", err)
		}
		return false
	}
	logger := s.core.Logger().With(logs.Any("task_id", task.ObjectID()))
	taskCtx := newTaskContext(ctx, task, logger)
	defer taskCtx.Close()
	factory, ok := registeredTasks[task.Kind()]
	if !ok {
		logger.Errorf("Unsupported task: %v", task.Kind())
		return true
	}
	impl := factory.New(s)
	logger.Info("Executing task", logs.Any("kind", task.Kind().String()))
	if err := impl.Execute(taskCtx); err != nil {
		s.core.Logger().Error("Task failed", err)
		statusCtx, cancel := context.WithTimeout(s.core.Context(), 30*time.Second)
		defer cancel()
		if err := task.SetStatus(statusCtx, models.FailedTask); err != nil {
			logger.Error("Unable to set failed task status", err)
		}
		return true
	}
	logger.Info("Task succeeded")
	statusCtx, cancel := context.WithTimeout(s.core.Context(), 30*time.Second)
	defer cancel()
	if err := task.SetStatus(statusCtx, models.SucceededTask); err != nil {
		logger.Error("Unable to set succeeded task status", err)
		return true
	}
	return true
}

func (s *Invoker) getSolution(ctx context.Context, id int64) (models.Solution, error) {
	solution, err := s.core.Solutions.Get(ctx, id)
	if err == sql.ErrNoRows {
		if err := s.core.Solutions.Sync(ctx); err != nil {
			return models.Solution{}, fmt.Errorf(
				"unable to sync solutions: %w", err,
			)
		}
		solution, err = s.core.Solutions.Get(ctx, id)
	}
	return solution, err
}

func readFile(name string, limit int) (string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	bytes := make([]byte, limit+1)
	read, err := file.Read(bytes)
	if err != nil && err != io.EOF {
		return "", err
	}
	if read > limit {
		return fixUTF8String(string(bytes[:limit])) + "...", nil
	}
	return fixUTF8String(string(bytes[:read])), nil
}

func fixUTF8String(s string) string {
	return strings.ReplaceAll(strings.ToValidUTF8(s, ""), "\u0000", "")
}

var (
	sqlRepeatableRead = gosql.WithIsolation(sql.LevelRepeatableRead)
)
