package core

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/labstack/gommon/log"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/config"
	"github.com/udovin/solve/internal/db"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/logs"
)

// Core manages all available resources.
type Core struct {
	// Config contains config.
	Config config.Config
	// Settings contains settings store.
	Settings *models.SettingStore
	// Tasks contains task store.
	Tasks *models.TaskStore
	// Locks contains lock store.
	Locks *models.LockStore
	// Files contains file store.
	Files models.FileStore
	// Roles contains role store.
	Roles *models.RoleStore
	// RoleEdges contains role edge store.
	RoleEdges *models.RoleEdgeStore
	// Accounts contains account store.
	Accounts *models.AccountStore
	// AccountRoles contains account role store.
	AccountRoles *models.AccountRoleStore
	// Sessions contains session store.
	Sessions *models.SessionStore
	// Tokens contains token store.
	Tokens *models.TokenStore
	// Users contains user store.
	Users *models.UserStore
	// Scopes contains scope store.
	Scopes *models.ScopeStore
	// ScopeUsers contains scope user store.
	ScopeUsers *models.ScopeUserStore
	// Problems contains problems store.
	Problems *models.ProblemStore
	// ProblemResources contains problem resources store.
	ProblemResources *models.ProblemResourceStore
	// Solutions contains solutions store.
	Solutions *models.SolutionStore
	// Contests contains contest store.
	Contests *models.ContestStore
	// ContestProblems contains contest problems store.
	ContestProblems *models.ContestProblemStore
	// ContestParticipants contains contest participants store.
	ContestParticipants *models.ContestParticipantStore
	// ContestSolutions contains contest solutions store.
	ContestSolutions *models.ContestSolutionStore
	// ContestMessages contains contest messages store.
	ContestMessages models.ContestMessageStore
	// Compilers contains compiler store.
	Compilers *models.CompilerStore
	// Visits contains visit store.
	Visits *models.VisitStore
	//
	context context.Context
	cancel  context.CancelFunc
	waiter  sync.WaitGroup
	//
	taskContext context.Context
	taskCancel  context.CancelFunc
	taskWaiter  sync.WaitGroup
	// DB stores database connection.
	DB *gosql.DB
	// logger contains logger.
	logger *logs.Logger
}

// NewCore creates core instance from config.
func NewCore(cfg config.Config) (*Core, error) {
	conn, err := cfg.DB.Create()
	if err != nil {
		return nil, err
	}
	logger := logs.NewLogger()
	logger.SetHeader(`{"time":"${time_rfc3339_nano}","level":"${level}"}`)
	logger.SetLevel(log.Lvl(cfg.LogLevel))
	return &Core{Config: cfg, DB: conn, logger: logger}, nil
}

// Logger returns logger instance.
func (c *Core) Logger() *logs.Logger {
	return c.logger
}

// Start starts application and data synchronization.
func (c *Core) Start() error {
	if c.cancel != nil {
		return fmt.Errorf("core already started")
	}
	c.Logger().Debug("Starting core")
	c.context, c.cancel = context.WithCancel(context.Background())
	c.taskContext, c.taskCancel = context.WithCancel(c.context)
	if err := c.startStoreLoops(); err != nil {
		c.Stop()
		return err
	}
	c.Logger().Debug("Core started")
	return nil
}

// Stop stops syncing stores.
func (c *Core) Stop() {
	if c.cancel == nil {
		return
	}
	c.Logger().Debug("Stopping core")
	defer c.Logger().Debug("Core stopped")
	c.taskCancel()
	c.taskWaiter.Wait()
	c.cancel()
	c.waiter.Wait()
	c.context, c.cancel = nil, nil
}

func (c *Core) Context() context.Context {
	return c.context
}

// WrapTx runs function with transaction.
func (c *Core) WrapTx(
	ctx context.Context, fn func(ctx context.Context) error,
	options ...gosql.BeginTxOption,
) (err error) {
	return gosql.WrapTx(ctx, c.DB, func(tx *sql.Tx) error {
		return fn(db.WithTx(ctx, tx))
	}, options...)
}

// StartTask starts task in new goroutine.
func (c *Core) StartTask(name string, task func(ctx context.Context)) {
	c.Logger().Info("Start task", logs.Any("task", name))
	c.taskWaiter.Add(1)
	c.startCoreTask(func() {
		defer c.taskWaiter.Done()
		defer c.Logger().Info("Task finished", logs.Any("task", name))
		task(c.taskContext)
	})
}

func (c *Core) startCoreTask(task func()) {
	c.waiter.Add(1)
	go func() {
		defer c.waiter.Done()
		task()
	}()
}

func (c *Core) StartUniqueTask(name string, task func(ctx context.Context)) {
	c.startCoreTask(func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		c.startUniqueTask(name, task)
		for {
			select {
			case <-c.taskContext.Done():
				return
			case <-ticker.C:
				c.startUniqueTask(name, task)
			}
		}
	})
}

func (c *Core) startUniqueTask(name string, task func(ctx context.Context)) {
	c.taskWaiter.Add(1)
	defer c.taskWaiter.Done()
	guard, err := c.Locks.AcquireByName(c.taskContext, name)
	if err != nil {
		if err == context.Canceled || err == models.ErrLockAcquired {
			return
		}
		if err == sql.ErrNoRows {
			lock := models.Lock{Name: name}
			if err := c.Locks.Create(c.taskContext, &lock); err != nil {
				c.Logger().Warn("Cannot create lock for task", logs.Any("task", name), err)
			}
			return
		}
		c.Logger().Warn("Cannot acquire lock for task", logs.Any("task", name), err)
		return
	}
	defer func() {
		if err := guard.Release(c.context); err != nil {
			c.Logger().Warn("Cannot release lock for task", logs.Any("task", name), err)
		}
	}()
	c.Logger().Info("Start unique task", logs.Any("task", name))
	defer c.Logger().Info("Unique task finished", logs.Any("task", name))
	waiter := sync.WaitGroup{}
	defer waiter.Wait()
	ctx, cancel := context.WithCancel(c.taskContext)
	defer cancel()
	waiter.Add(1)
	go func() {
		defer waiter.Done()
		defer cancel()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := guard.Ping(ctx); err != nil {
					if err != context.Canceled {
						c.Logger().Warn("Cannot ping lock for task", logs.Any("task", name), err)
					}
					return
				}
				c.Logger().Debug("Pinged lock for task", logs.Any("task", name), err)
			}

		}
	}()
	task(ctx)
}
