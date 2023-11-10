package core

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

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
