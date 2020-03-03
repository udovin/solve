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

// App manages all available resources.
type App struct {
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

// NewApp creates app instance from config.
func NewApp(cfg config.Config) (*App, error) {
	conn, err := cfg.DB.Create()
	if err != nil {
		return nil, err
	}
	return &App{db: conn, Config: cfg}, nil
}

func (a *App) startManagers(start func(models.Manager, time.Duration)) {
	start(a.Actions, time.Second)
	start(a.Roles, time.Minute)
	start(a.RoleEdges, time.Minute)
	start(a.Users, time.Second)
	start(a.UserFields, time.Second)
	start(a.UserRoles, time.Minute)
	start(a.Sessions, time.Second)
}

// SetupInvokerManagers prepares managers for running invoker.
func (a *App) SetupInvokerManagers() {}

// SetupAllManagers prepares all managers.
func (a *App) SetupAllManagers() error {
	salt, err := a.Config.Security.PasswordSalt.Secret()
	if err != nil {
		return err
	}
	dialect := GetDialect(a.Config.DB.Driver)
	a.Actions = models.NewActionManager(
		"solve_action", "solve_action_event", dialect,
	)
	a.Roles = models.NewRoleManager(
		"solve_role", "solve_role_event", dialect,
	)
	a.RoleEdges = models.NewRoleEdgeManager(
		"solve_role_edge", "solve_role_edge_event", dialect,
	)
	a.Users = models.NewUserManager(
		"solve_user", "solve_user_event", salt, dialect,
	)
	a.UserFields = models.NewUserFieldManager(
		"solve_user_field", "solve_user_field_event", dialect,
	)
	a.UserRoles = models.NewUserRoleManager(
		"solve_user_role", "solve_user_role_event", dialect,
	)
	a.Sessions = models.NewSessionManager(
		"solve_session", "solve_session_event", dialect,
	)
	a.Visits = models.NewVisitManager("solve_visit", dialect)
	return nil
}

// WithTx runs function with transaction.
func (a *App) WithTx(fn func(*sql.Tx) error) (err error) {
	var tx *sql.Tx
	if tx, err = a.db.Begin(); err != nil {
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

// var guestRoles = []string{
// 	models.AuthStatusRole,
// 	models.RegisterRole,
// }

func (a *App) getGroupRoles(id int64) (Roles, error) {
	stack := []int64{id}
	roles := Roles{}
	for len(stack) > 0 {
		roleID := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		edges, err := a.RoleEdges.FindByRole(roleID)
		if err != nil {
			return nil, err
		}
		for _, edge := range edges {
			role, err := a.Roles.Get(edge.ChildID)
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
func (a *App) GetGuestRoles() (Roles, error) {
	role, err := a.Roles.GetByCode(models.GuestRoleGroup)
	if err != nil {
		return Roles{}, err
	}
	return a.getGroupRoles(role.ID)
}

// var userRoles = []string{
// 	models.AuthStatusRole,
// 	models.LoginRole,
// 	models.LogoutRole,
// }

// GetUserRoles returns roles for user.
func (a *App) GetUserRoles(id int64) (Roles, error) {
	role, err := a.Roles.GetByCode(models.UserRoleGroup)
	if err != nil {
		return Roles{}, err
	}
	return a.getGroupRoles(role.ID)
}

// HasRole return true if role set has this role or parent role.
func (a *App) HasRole(roles Roles, code string) (bool, error) {
	role, err := a.Roles.GetByCode(code)
	if err != nil {
		return false, err
	}
	if _, ok := roles[role.ID]; ok {
		return true, nil
	}
	return false, nil
}

// Start starts application and data synchronization.
func (a *App) Start() error {
	if a.closer != nil {
		return fmt.Errorf("app already started")
	}
	a.closer = make(chan struct{})
	errs := make(chan error)
	count := 0
	a.startManagers(func(m models.Manager, d time.Duration) {
		v := reflect.ValueOf(m)
		if m == nil || (v.Kind() == reflect.Ptr && v.IsNil()) {
			return
		}
		count++
		a.waiter.Add(1)
		go a.startManager(m, d, errs)
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
func (a *App) Stop() {
	if a.closer == nil {
		return
	}
	close(a.closer)
	a.waiter.Wait()
	a.closer = nil
}

func (a *App) startManager(
	m models.Manager, d time.Duration, errs chan<- error,
) {
	defer a.waiter.Done()
	err := a.WithTx(m.InitTx)
	errs <- err
	if err != nil {
		return
	}
	ticker := time.NewTicker(d)
	for {
		select {
		case <-ticker.C:
			if err := a.WithTx(m.SyncTx); err != nil {
				log.Println("Error:", err)
			}
		case <-a.closer:
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
