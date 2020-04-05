package core

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

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
	// Users contains user manager.
	Users *models.UserManager
	// UserFields contains user field manager.
	UserFields *models.UserFieldManager
	// UserRoles contains user role manager.
	UserRoles *models.UserRoleManager
	// Sessions contains session manager.
	Sessions *models.SessionManager
	// Visits contains visit manager.
	Visits *models.VisitManager
	// Stores.
	Compilers       *models.CompilerStore
	Problems        *models.ProblemStore
	Statements      *models.StatementStore
	Solutions       *models.SolutionStore
	Reports         *models.ReportStore
	Contests        *models.ContestStore
	ContestProblems *models.ContestProblemStore
	Participants    *models.ParticipantStore
	closer          chan struct{}
	waiter          sync.WaitGroup
	// db store database connection.
	db *sql.DB
}

// NewCore creates core instance from config.
func NewCore(cfg config.Config) (*Core, error) {
	conn, err := cfg.DB.Create()
	if err != nil {
		return nil, err
	}
	return &Core{db: conn, Config: cfg}, nil
}

func (c *Core) startManagers(start func(models.Manager, time.Duration)) {
	start(c.Actions, time.Second)
	start(c.Roles, time.Minute)
	start(c.RoleEdges, time.Minute)
	start(c.Users, time.Second)
	start(c.UserFields, time.Second)
	start(c.UserRoles, time.Minute)
	start(c.Sessions, time.Second)
}

// SetupInvokerManagers prepares managers for running invoker.
func (c *Core) SetupInvokerManagers() {}

// SetupAllManagers prepares all managers.
func (c *Core) SetupAllManagers() error {
	salt, err := c.Config.Security.PasswordSalt.Secret()
	if err != nil {
		return err
	}
	dialect := GetDialect(c.Config.DB.Driver)
	c.Actions = models.NewActionManager(
		"solve_action", "solve_action_event", dialect,
	)
	c.Roles = models.NewRoleManager(
		"solve_role", "solve_role_event", dialect,
	)
	c.RoleEdges = models.NewRoleEdgeManager(
		"solve_role_edge", "solve_role_edge_event", dialect,
	)
	c.Users = models.NewUserManager(
		"solve_user", "solve_user_event", salt, dialect,
	)
	c.UserFields = models.NewUserFieldManager(
		"solve_user_field", "solve_user_field_event", dialect,
	)
	c.UserRoles = models.NewUserRoleManager(
		"solve_user_role", "solve_user_role_event", dialect,
	)
	c.Sessions = models.NewSessionManager(
		"solve_session", "solve_session_event", dialect,
	)
	c.Visits = models.NewVisitManager("solve_visit", dialect)
	return nil
}

// WithTx runs function with transaction.
func (c *Core) WithTx(fn func(*sql.Tx) error) (err error) {
	var tx *sql.Tx
	if tx, err = c.db.Begin(); err != nil {
		return err
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

// Roles contains roles.
type Roles map[int64]struct{}

// getGroupRoles returns roles for group with specified ID.
func (c *Core) getGroupRoles(id int64) (Roles, error) {
	stack := []int64{id}
	roles := Roles{}
	for len(stack) > 0 {
		roleID := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		edges, err := c.RoleEdges.FindByRole(roleID)
		if err != nil {
			return nil, err
		}
		for _, edge := range edges {
			role, err := c.Roles.Get(edge.ChildID)
			if err != nil {
				return nil, err
			}
			if _, ok := roles[role.ID]; !ok {
				roles[role.ID] = struct{}{}
				stack = append(stack, role.ID)
			}
		}
	}
	return roles, nil
}

// GetGuestRoles returns roles for guest account.
func (c *Core) GetGuestRoles() (Roles, error) {
	role, err := c.Roles.GetByCode(models.GuestGroupRole)
	if err != nil {
		return Roles{}, err
	}
	return c.getGroupRoles(role.ID)
}

// GetUserRoles returns roles for user.
func (c *Core) GetUserRoles(id int64) (Roles, error) {
	role, err := c.Roles.GetByCode(models.UserGroupRole)
	if err != nil {
		return Roles{}, err
	}
	return c.getGroupRoles(role.ID)
}

// HasRole return true if role set has this role or parent role.
func (c *Core) HasRole(roles Roles, code string) (bool, error) {
	role, err := c.Roles.GetByCode(code)
	if err != nil {
		return false, err
	}
	_, ok := roles[role.ID]
	return ok, nil
}

// Start starts application and data synchronization.
func (c *Core) Start() error {
	if c.closer != nil {
		return fmt.Errorf("app already started")
	}
	c.closer = make(chan struct{})
	errs := make(chan error)
	count := 0
	c.startManagers(func(m models.Manager, d time.Duration) {
		v := reflect.ValueOf(m)
		if m == nil || (v.Kind() == reflect.Ptr && v.IsNil()) {
			return
		}
		count++
		c.waiter.Add(1)
		go c.startManager(m, d, errs)
	})
	var err error
	for i := 0; i < count; i++ {
		lastErr := <-errs
		if lastErr != nil {
			log.Println("Error:", lastErr)
			err = lastErr
		}
	}
	return err
}

// Stop stops syncing stores.
func (c *Core) Stop() {
	if c.closer == nil {
		return
	}
	close(c.closer)
	c.waiter.Wait()
	c.closer = nil
}

func (c *Core) startManager(
	m models.Manager, d time.Duration, errs chan<- error,
) {
	defer c.waiter.Done()
	err := c.WithTx(m.InitTx)
	errs <- err
	if err != nil {
		return
	}
	ticker := time.NewTicker(d)
	for {
		select {
		case <-ticker.C:
			if err := c.WithTx(m.SyncTx); err != nil {
				log.Println("Error:", err)
			}
		case <-c.closer:
			ticker.Stop()
			return
		}
	}
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
