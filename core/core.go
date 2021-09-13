package core

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/labstack/gommon/log"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/models"
)

// Core manages all available resources.
type Core struct {
	// Config contains config.
	Config config.Config
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
	// UserFields contains user field store.
	UserFields *models.UserFieldStore
	// Problems contains problems store.
	Problems *models.ProblemStore
	// Solutions contains solutions store.
	Solutions *models.SolutionStore
	// Contests contains contest store.
	Contests *models.ContestStore
	// ContestProblems contains contest problems store.
	ContestProblems *models.ContestProblemStore
	// Visits contains visit store.
	Visits *models.VisitStore
	//
	context context.Context
	cancel  context.CancelFunc
	waiter  sync.WaitGroup
	// db stores database connection.
	DB *sql.DB
	// logger contains logger.
	logger *log.Logger
}

// NewCore creates core instance from config.
func NewCore(cfg config.Config) (*Core, error) {
	conn, err := cfg.DB.Create()
	if err != nil {
		return nil, err
	}
	logger := log.New("core")
	logger.SetLevel(cfg.LogLevel)
	logger.EnableColor()
	return &Core{Config: cfg, DB: conn, logger: logger}, nil
}

// Logger returns logger instance.
func (c *Core) Logger() *log.Logger {
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

// WithTx runs function with transaction.
func (c *Core) WithTx(
	ctx context.Context, fn func(tx *sql.Tx) error,
) (err error) {
	var tx *sql.Tx
	if tx, err = c.DB.BeginTx(ctx, nil); err != nil {
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()
	return fn(tx)
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

// Dialect returns dialect of core DB.
func (c *Core) Dialect() db.Dialect {
	return GetDialect(c.Config.DB.Driver)
}

// GetDialect returns SQL dialect from database driver.
func GetDialect(driver config.DBDriver) db.Dialect {
	switch driver {
	case config.PostgresDriver:
		return db.Postgres
	default:
		return db.SQLite
	}
}
