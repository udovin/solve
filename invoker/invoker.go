// Package invoker represents solve implementation for running asynchronious
// tasks like compiling and judging solutions.
package invoker

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

// Invoker represents manager for asynchronous actions (invocations).
type Invoker struct {
	core    *core.Core
	files   *managers.FileManager
	factory *factory
}

// New creates a new instance of Invoker.
func New(core *core.Core) *Invoker {
	s := Invoker{
		core: core,
	}
	if core.Config.Storage != nil {
		s.files = managers.NewFileManager(core)
	}
	return &s
}

// Start starts invoker daemons.
//
// This function will spawn config.Invoker.Workers amount of goroutines.
func (s *Invoker) Start() error {
	if s.factory != nil {
		return fmt.Errorf("factory already created")
	}
	factory, err := newFactory("/tmp/containers")
	if err != nil {
		return err
	}
	s.factory = factory
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
	logger := s.core.Logger().With(core.Any("task_id", task.ObjectID()))
	taskCtx := newTaskContext(ctx, task, logger)
	defer taskCtx.Close()
	factory, ok := registeredTasks[task.Kind()]
	if !ok {
		logger.Errorf("Unsupported task: %v", task.Kind())
		return true
	}
	impl := factory.New(s)
	logger.Info("Executing task", core.Any("kind", task.Kind().String()))
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
	solution, err := s.core.Solutions.Get(id)
	if err == sql.ErrNoRows {
		if err := s.core.Solutions.Sync(ctx); err != nil {
			return models.Solution{}, fmt.Errorf(
				"unable to sync solutions: %w", err,
			)
		}
		solution, err = s.core.Solutions.Get(id)
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
	if err != nil {
		return "", err
	}
	if read > limit {
		return strings.ToValidUTF8(string(bytes[:limit]), "") + "...", nil
	}
	return strings.ToValidUTF8(string(bytes[:read]), ""), nil
}

func compareFiles(outputPath, answerPath string) (string, bool, error) {
	output, err := ioutil.ReadFile(outputPath)
	if err != nil {
		return "", false, err
	}
	answer, err := ioutil.ReadFile(answerPath)
	if err != nil {
		return "", false, err
	}
	outputStr := string(output)
	outputStr = strings.ReplaceAll(outputStr, "\n", "")
	outputStr = strings.ReplaceAll(outputStr, "\r", "")
	outputStr = strings.ReplaceAll(outputStr, "\t", "")
	outputStr = strings.ReplaceAll(outputStr, " ", "")
	answerStr := string(answer)
	answerStr = strings.ReplaceAll(answerStr, "\n", "")
	answerStr = strings.ReplaceAll(answerStr, "\r", "")
	answerStr = strings.ReplaceAll(answerStr, "\t", "")
	answerStr = strings.ReplaceAll(answerStr, " ", "")
	if outputStr == answerStr {
		return "ok", true, nil
	} else {
		if len(output) > 100 {
			output = output[:100]
		}
		if len(answer) > 100 {
			answer = answer[:100]
		}
		return fmt.Sprintf("expected %q, got %q", string(answer), string(output)), false, nil
	}
}

var (
	sqlRepeatableRead = gosql.WithIsolation(sql.LevelRepeatableRead)
)
