// Package invoker represents solve implementation for running asynchronious
// tasks like compiling and judging solutions.
package invoker

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/core"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"

	"github.com/udovin/solve/internal/pkg/logs"
	"github.com/udovin/solve/internal/pkg/safeexec"

	compilerCache "github.com/udovin/solve/internal/pkg/compilers/cache"
	problemCache "github.com/udovin/solve/internal/pkg/problems/cache"
)

// Invoker represents manager for asynchronous actions (invocations).
type Invoker struct {
	core            *core.Core
	files           *managers.FileManager
	solutions       *managers.SolutionManager
	compilerImages  *compilerCache.CompilerImageManager
	problemPackages *problemCache.ProblemPackageManager
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
	cgroupPath := safeexecConfig.Cgroup
	if len(cgroupPath) == 0 {
		cgroupPath = "../solve-safeexec"
	}
	var safeexecOptions []safeexec.Option
	if safeexecConfig.MemoryPeak == nil || *safeexecConfig.MemoryPeak {
		safeexecOptions = append(safeexecOptions, safeexec.WithMemoryPeak)
	}
	safeexec, err := safeexec.NewManager(
		safeexecConfig.Path, "/tmp/solve-safeexec", cgroupPath, safeexecOptions...,
	)
	if err != nil {
		return err
	}
	s.compilerImages, err = compilerCache.NewCompilerImageManager(
		s.files, safeexec, "/tmp/solve-compilers",
	)
	if err != nil {
		return err
	}
	s.problemPackages, err = problemCache.NewProblemPackageManager(s.files, "/tmp/solve-problems")
	if err != nil {
		return err
	}
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

var (
	sqlRepeatableRead = gosql.WithIsolation(sql.LevelRepeatableRead)
)
