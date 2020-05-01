package core

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/models"
)

// Core manages all available resources.
type Core struct {
	// Config contains config.
	Config config.Config
	// Actions contains action manager.
	Actions *models.ActionManager
	// Roles contains role manager.
	Roles *models.RoleManager
	// RoleEdges contains role edge manager.
	RoleEdges *models.RoleEdgeManager
	// Accounts contains account manager.
	Accounts *models.AccountManager
	// AccountRoles contains account role manager.
	AccountRoles *models.AccountRoleManager
	// Sessions contains session manager.
	Sessions *models.SessionManager
	// Users contains user manager.
	Users *models.UserManager
	// UserFields contains user field manager.
	UserFields *models.UserFieldManager
	// Problems contains problems manager.
	Problems *models.ProblemManager
	// Contests contains contest manager.
	Contests *models.ContestManager
	// ContestProblems contains contest problems manager.
	ContestProblems *models.ContestProblemManager
	// Visits contains visit manager.
	Visits *models.VisitManager
	//
	context context.Context
	cancel  context.CancelFunc
	waiter  sync.WaitGroup
	// db store database connection.
	db *sql.DB
}

// NewCore creates core instance from config.
func NewCore(cfg config.Config) (*Core, error) {
	conn, err := cfg.DB.Create()
	if err != nil {
		return nil, err
	}
	return &Core{Config: cfg, db: conn}, nil
}

// Start starts application and data synchronization.
func (c *Core) Start() error {
	if c.cancel != nil {
		return fmt.Errorf("core already started")
	}
	c.context, c.cancel = context.WithCancel(context.Background())
	return c.startManagerLoops()
}

// Stop stops syncing stores.
func (c *Core) Stop() {
	if c.cancel == nil {
		return
	}
	c.cancel()
	c.waiter.Wait()
	c.context, c.cancel = nil, nil
}

// WithTx runs function with transaction.
func (c *Core) WithTx(
	ctx context.Context, fn func(tx *sql.Tx) error,
) (err error) {
	var tx *sql.Tx
	if tx, err = c.db.BeginTx(ctx, nil); err != nil {
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
	c.waiter.Add(1)
	go func() {
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
