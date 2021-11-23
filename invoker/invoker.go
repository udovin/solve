package invoker

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/opencontainers/runc/libcontainer"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg"

	_ "github.com/opencontainers/runc/libcontainer/nsenter"
)

// Invoker represents manager for asynchronous actions (invocations).
type Invoker struct {
	core    *core.Core
	factory libcontainer.Factory
}

// New creates a new instance of Invoker.
func New(c *core.Core) *Invoker {
	return &Invoker{core: c}
}

// Start starts invoker daemons.
//
// This function will spawn config.Invoker.Threads amount of goroutines.
func (s *Invoker) Start() error {
	if s.factory != nil {
		return fmt.Errorf("factory already created")
	}
	factory, err := libcontainer.New("")
	if err != nil {
		return err
	}
	s.factory = factory
	threads := s.core.Config.Invoker.Threads
	if threads <= 0 {
		threads = 1
	}
	for i := 0; i < threads; i++ {
		s.core.StartTask(s.runDaemon)
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
	var task models.Task
	if err := s.core.WithTx(ctx, func(tx *sql.Tx) error {
		var err error
		task, err = s.core.Tasks.PopQueuedTx(tx)
		return err
	}); err != nil {
		if err != sql.ErrNoRows {
			s.core.Logger().Error("Error:", err)
		}
		return false
	}
	defer func() {
		if r := recover(); r != nil {
			task.Status = models.Failed
			s.core.Logger().Error("Task panic:", r)
		}
		ctx, cancel := context.WithDeadline(context.Background(), time.Unix(task.ExpireTime, 0))
		defer cancel()
		if err := s.core.WithTx(ctx, func(tx *sql.Tx) error {
			return s.core.Tasks.UpdateTx(tx, task)
		}); err != nil {
			s.core.Logger().Error("Error:", err)
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
					s.core.Logger().Error("Task expired:", task.ID)
					return
				}
				clone := task
				if err := s.core.WithTx(ctx, func(tx *sql.Tx) error {
					clone.ExpireTime = time.Now().Add(5 * time.Second).Unix()
					return s.core.Tasks.UpdateTx(tx, clone)
				}); err != nil {
					s.core.Logger().Warn("Unable to ping task:", err)
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
		s.core.Logger().Error("Task failed: ", err)
		task.Status = models.Failed
	} else {
		task.Status = models.Succeeded
	}
	return true
}

func (s *Invoker) onTask(ctx context.Context, task models.Task) error {
	s.core.Logger().Debug("Received new task: ", task.ID)
	switch task.Kind {
	case models.JudgeSolution:
		return s.onJudgeSolution(ctx, task)
	default:
		s.core.Logger().Error("Unknown task: ", task.Kind)
		return fmt.Errorf("unknown task")
	}
}

func (s *Invoker) onJudgeSolution(ctx context.Context, task models.Task) error {
	var taskConfig models.JudgeSolutionConfig
	if err := task.ScanConfig(&taskConfig); err != nil {
		return fmt.Errorf("unable to scan task config: %w", err)
	}
	solution, err := s.core.Solutions.Get(taskConfig.SolutionID)
	if err != nil {
		return fmt.Errorf("unable to fetch task solution: %w", err)
	}
	problem, err := s.core.Problems.Get(solution.ProblemID)
	if err != nil {
		return fmt.Errorf("unable to fetch task problem: %w", err)
	}
	tempDir, err := makeTempDir()
	if err != nil {
		return err
	}
	s.core.Logger().Info(tempDir)
	// defer func() {
	// 	_ = os.RemoveAll(tempDir)
	// }()
	problemPath := path.Join(
		s.core.Config.Storage.ProblemsDir,
		fmt.Sprintf("%d.zip", problem.ID),
	)
	compierPath := path.Join(
		s.core.Config.Storage.CompilersDir,
		"dosbox-tasm.tar.gz",
	)
	compilerRootFS := path.Join(tempDir, "rootfs")
	if err := pkg.ExtractTarGz(compierPath, compilerRootFS); err != nil {
		return err
	}
	s.core.Logger().Info(compilerRootFS)
	_ = problemPath
	return fmt.Errorf("not implemented")
}

func makeTempDir() (string, error) {
	for i := 0; i < 100; i++ {
		name, err := uuid.NewV4()
		if err != nil {
			return "", err
		}
		dirPath := path.Join(os.TempDir(), name.String())
		if err := os.MkdirAll(dirPath, 0777); err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", err
		}
		return dirPath, nil
	}
	return "", fmt.Errorf("unable to create temp directory")
}

func init() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runtime.GOMAXPROCS(1)
		runtime.LockOSThread()
		factory, err := libcontainer.New("")
		if err != nil {
			log.Fatal(err)
		}
		if err := factory.StartInitialization(); err != nil {
			log.Fatal(err)
		}
		panic("--this line should have never been executed, congratulations--")
	}
}
