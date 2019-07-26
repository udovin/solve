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
	Users       *models.UserStore
	Sessions    *models.SessionStore
	Problems    *models.ProblemStore
	Contests    *models.ContestStore
	Roles       *models.RoleStore
	Permissions *models.PermissionStore
	// Used for regularly updating stores
	ticker *time.Ticker
	// Password salt
	PasswordSalt string
}

// Create solve app from config
func NewApp(cfg *config.Config) (*App, error) {
	// Try to create database connection pool
	db, err := cfg.Database.CreateDB()
	if err != nil {
		return nil, err
	}
	app := App{
		Config: *cfg,
		Users: models.NewUserStore(
			db, "solve_user", "solve_user_change",
		),
		Sessions: models.NewSessionStore(
			db, "solve_session", "solve_session_change",
		),
		Problems: models.NewProblemStore(
			db, "solve_problem", "solve_problem_change",
		),
		Contests: models.NewContestStore(
			db, "solve_contest", "solve_contest_change",
		),
		Roles: models.NewRoleStore(
			db, "solve_role", "solve_role_change",
		),
		Permissions: models.NewPermissionStore(
			db, "solve_permission", "solve_permission_change",
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

// TODO: Some tables can be large and some other are not large,
//   so we should update all tables fully asynchronously.
func (a *App) Start() {
	a.syncStoresTick()
	// Update all store at most in one second. If exists a store which
	// required more than one second for sync, we will slow down all
	// other store syncs.
	a.ticker = time.NewTicker(time.Second)
	go a.syncStores()
}

// Stop syncing stores
// TODO: Stop should hanging current goroutine while some syncs are running
func (a *App) Stop() {
	a.ticker.Stop()
}

// Almost infinite loop of syncing stores
func (a *App) syncStores() {
	for range a.ticker.C {
		a.syncStoresTick()
	}
}

// Sync all stores with database state
func (a *App) syncStoresTick() {
	wg := sync.WaitGroup{}
	go a.runManagerSync(wg, a.Users.Manager)
	go a.runManagerSync(wg, a.Sessions.Manager)
	go a.runManagerSync(wg, a.Problems.Manager)
	go a.runManagerSync(wg, a.Contests.Manager)
	go a.runManagerSync(wg, a.Roles.Manager)
	go a.runManagerSync(wg, a.Permissions.Manager)
	wg.Wait()
}

// Sync store with database
// This method is created for running in separate goroutine, so we
// should pass WaitGroup to understand that goroutine is finished.
func (a *App) runManagerSync(wg sync.WaitGroup, m *models.ChangeManager) {
	wg.Add(1)
	defer wg.Done()
	if err := m.Sync(); err != nil {
		log.Print("unable to sync store: ", err)
	}
}
