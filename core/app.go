package core

import (
	"log"
	"sync"
	"time"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/models"
)

// App manages all available resources
type App struct {
	Config config.Config
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
	closer          chan struct{}
	waiter          sync.WaitGroup
	// Password salt
	PasswordSalt string
}

// NewApp creates app instance from config
func NewApp(cfg *config.Config) (*App, error) {
	// Try to create database connection pool
	db, err := cfg.DB.Create()
	if err != nil {
		return nil, err
	}
	app := App{
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
	}
	// We do not want to load value every time
	// in case of FileSecret or EnvSecret
	app.PasswordSalt, err = cfg.Security.PasswordSalt.GetValue()
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// Start starts application and data synchronization
func (a *App) Start() error {
	a.closer = make(chan struct{})
	errs := make(chan error)
	defer close(errs)
	stores := 0
	runManagerSync := func(m *models.ChangeManager) {
		stores++
		go a.runManagerSync(m, errs)
	}
	runManagerSync(a.Users.Manager)
	runManagerSync(a.UserFields.Manager)
	runManagerSync(a.Sessions.Manager)
	runManagerSync(a.Compilers.Manager)
	runManagerSync(a.Problems.Manager)
	runManagerSync(a.Statements.Manager)
	runManagerSync(a.Solutions.Manager)
	runManagerSync(a.Reports.Manager)
	runManagerSync(a.Contests.Manager)
	runManagerSync(a.ContestProblems.Manager)
	var err error
	for i := 0; i < stores; i++ {
		lastErr := <-errs
		if lastErr != nil {
			log.Println("error:", lastErr)
			err = lastErr
		}
	}
	if err != nil {
		a.Stop()
	}
	return err
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
