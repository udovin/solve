package core

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/labstack/gommon/log"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/config"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/models"
)

// Core manages all available resources.
type Core struct {
	// Config contains config.
	Config config.Config
	// Settings contains settings store.
	Settings *models.SettingStore
	// Tasks contains task store.
	Tasks *models.TaskStore
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
	// Users contains user store.
	Users *models.UserStore
	// Problems contains problems store.
	Problems *models.ProblemStore
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
	// Compilers contains compiler store.
	Compilers *models.CompilerStore
	// Visits contains visit store.
	Visits *models.VisitStore
	//
	context context.Context
	cancel  context.CancelFunc
	waiter  sync.WaitGroup
	// DB stores database connection.
	DB *gosql.DB
	// logger contains logger.
	logger *Logger
}

// NewCore creates core instance from config.
func NewCore(cfg config.Config) (*Core, error) {
	conn, err := cfg.DB.Create()
	if err != nil {
		return nil, err
	}
	logger := Logger{Logger: log.New("")}
	logger.SetHeader(`{"time":"${time_rfc3339_nano}","level":"${level}"}`)
	logger.SetLevel(log.Lvl(cfg.LogLevel))
	return &Core{Config: cfg, DB: conn, logger: &logger}, nil
}

// Logger returns logger instance.
func (c *Core) Logger() *Logger {
	return c.logger
}

// Start starts application and data synchronization.
func (c *Core) Start() error {
	if c.cancel != nil {
		return fmt.Errorf("core already started")
	}
	c.Logger().Debug("Starting core")
	defer c.Logger().Debug("Core started")
	c.context, c.cancel = context.WithCancel(context.Background())
	return c.startStoreLoops()
}

// Stop stops syncing stores.
func (c *Core) Stop() {
	if c.cancel == nil {
		return
	}
	c.Logger().Debug("Stopping core")
	defer c.Logger().Debug("Core stopped")
	c.cancel()
	c.waiter.Wait()
	c.context, c.cancel = nil, nil
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
func (c *Core) StartTask(task func(ctx context.Context)) {
	c.Logger().Debug("Start core task")
	c.waiter.Add(1)
	go func() {
		defer c.Logger().Debug("Core task finished")
		defer c.waiter.Done()
		task(c.context)
	}()
}
