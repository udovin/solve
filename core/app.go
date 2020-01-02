package core

import (
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/models"
)

// App manages all available resources
type App struct {
	db *sql.DB
	// Config contains config
	Config config.Config
	// Actions contains action manager
	Actions *models.ActionManager
	// Roles contains role manager
	Roles *models.RoleManager
	// UserRoles contains user role manager
	UserRoles *models.UserRoleManager
	// Visits contains visit manager
	Visits *models.VisitManager
	// Stores
	Users           *models.UserStore
	UserFields      *models.UserFieldStore
	Sessions        *models.SessionStore
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
	// Password salt
	PasswordSalt string
}

func getDialect(driver config.DatabaseDriver) db.Dialect {
	switch driver {
	case config.PostgresDriver:
		return db.Postgres
	default:
		return db.SQLite
	}
}

// NewApp creates app instance from config
func NewApp(cfg *config.Config) (*App, error) {
	// Try to create database connection pool
	db, err := cfg.DB.Create()
	if err != nil {
		return nil, err
	}
	app := App{
		db:     db,
		Config: *cfg,
		Users: models.NewUserStore(
			db, "solve_user", "solve_user_change",
		),
		UserFields: models.NewUserFieldStore(
			db, "solve_user_field", "solve_user_field_change",
		),
		Sessions: models.NewSessionStore(
			db, "solve_session", "solve_session_change",
		),
		Compilers: models.NewCompilerStore(
			db, "solve_compiler", "solve_compiler_change",
		),
		Problems: models.NewProblemStore(
			db, "solve_problem", "solve_problem_change",
		),
		Statements: models.NewStatementStore(
			db, "solve_statement", "solve_statement_change",
		),
		Solutions: models.NewSolutionStore(
			db, "solve_solution", "solve_solution_change",
		),
		Reports: models.NewReportStore(
			db, "solve_report", "solve_report_change",
		),
		Contests: models.NewContestStore(
			db, "solve_contest", "solve_contest_change",
		),
		ContestProblems: models.NewContestProblemStore(
			db, "solve_contest_problem", "solve_contest_problem_change",
		),
		Participants: models.NewParticipantStore(
			db, "solve_participant", "solve_participant_change",
		),
	}
	// We do not want to load value every time
	// in case of FileSecret or EnvSecret
	app.PasswordSalt, err = cfg.Security.PasswordSalt.Secret()
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// SetupInvokerManagers prepares managers for running invoker
func (a *App) SetupInvokerManagers() {

}

// SetupAllManagers prepares all managers
func (a *App) SetupAllManagers() {
	dialect := getDialect(a.Config.DB.Driver)
	a.Actions = models.NewActionManager(
		"solve_action", "solve_action_event", dialect,
	)
	a.Roles = models.NewRoleManager(
		"solve_role", "solve_role_event", dialect,
	)
	a.UserRoles = models.NewUserRoleManager(
		"solve_user_role", "solve_user_role_event", dialect,
	)
	a.Visits = models.NewVisitManager(a.db, "solve_visits", dialect)
}

// Start starts application and data synchronization
func (a *App) Start() error {
	a.closer = make(chan struct{})
	errs := make(chan error)
	defer close(errs)
	stores := 0
	runManager := func(m *models.ChangeManager) {
		stores++
		go a.runManagerSync(m, errs)
	}
	a.runManagers(runManager)
	var err error
	for i := 0; i < stores; i++ {
		lastErr := <-errs
		if lastErr != nil {
			log.Println("Error:", lastErr)
			err = lastErr
		}
	}
	if err != nil {
		a.Stop()
	}
	return err
}

func (a *App) runManagers(runManager func(m *models.ChangeManager)) {
	if a.Users != nil {
		runManager(a.Users.Manager)
	}
	if a.UserFields != nil {
		runManager(a.UserFields.Manager)
	}
	if a.Sessions != nil {
		runManager(a.Sessions.Manager)
	}
	if a.Compilers != nil {
		runManager(a.Compilers.Manager)
	}
	if a.Problems != nil {
		runManager(a.Problems.Manager)
	}
	if a.Statements != nil {
		runManager(a.Statements.Manager)
	}
	if a.Solutions != nil {
		runManager(a.Solutions.Manager)
	}
	if a.Reports != nil {
		runManager(a.Reports.Manager)
	}
	if a.Contests != nil {
		runManager(a.Contests.Manager)
	}
	if a.ContestProblems != nil {
		runManager(a.ContestProblems.Manager)
	}
	if a.Participants != nil {
		runManager(a.Participants.Manager)
	}
}

// Stop stops syncing stores
func (a *App) Stop() {
	close(a.closer)
	// Wait for all manager syncs to finish
	a.waiter.Wait()
}

// runManagerSync syncs store with database
func (a *App) runManagerSync(m *models.ChangeManager, errs chan<- error) {
	a.waiter.Add(1)
	defer a.waiter.Done()
	errs <- m.Init()
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-a.closer:
			ticker.Stop()
			return
		case <-ticker.C:
			if err := m.Sync(); err != nil {
				log.Println("Error:", err)
			}
		}
	}
}
