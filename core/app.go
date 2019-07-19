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
	UserStore       *models.UserStore
	SessionStore    *models.SessionStore
	ProblemStore    *models.ProblemStore
	ContestStore    *models.ContestStore
	RoleStore       *models.RoleStore
	PermissionStore *models.PermissionStore
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
		UserStore: models.NewUserStore(
			db, "solve_user", "solve_user_change",
		),
		SessionStore: models.NewSessionStore(
			db, "solve_session", "solve_session_change",
		),
		ProblemStore: models.NewProblemStore(
			db, "solve_problem", "solve_problem_change",
		),
		ContestStore: models.NewContestStore(
			db, "solve_contest", "solve_contest_change",
		),
		RoleStore: models.NewRoleStore(
			db, "solve_role", "solve_role_change",
		),
		PermissionStore: models.NewPermissionStore(
			db, "solve_permission", "solve_permission_change",
		),
	}
	// We do not want to load value every time
	// in case of FileSecret or VariableSecret
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
	go a.runManagerSync(wg, a.UserStore.Manager)
	go a.runManagerSync(wg, a.SessionStore.Manager)
	go a.runManagerSync(wg, a.ProblemStore.Manager)
	go a.runManagerSync(wg, a.ContestStore.Manager)
	go a.runManagerSync(wg, a.RoleStore.Manager)
	go a.runManagerSync(wg, a.PermissionStore.Manager)
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
