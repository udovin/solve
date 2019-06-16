package core

import (
	"log"
	"sync"
	"time"

	"../config"
	"../models"
)

type App struct {
	Config config.Config
	// Stores
	UserStore       *models.UserStore
	SessionStore    *models.SessionStore
	ProblemStore    *models.ProblemStore
	PermissionStore *models.PermissionStore
	RoleStore       *models.RoleStore
	// Used for regularly updating stores
	ticker *time.Ticker
}

func NewApp(cfg *config.Config) (*App, error) {
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
		PermissionStore: models.NewPermissionStore(
			db, "solve_permission", "solve_permission_change",
		),
		RoleStore: models.NewRoleStore(
			db, "solve_role", "solve_role_change",
		),
	}
	return &app, nil
}

func (a *App) Start() {
	a.ticker = time.NewTicker(time.Second)
	go a.syncStores()
}

func (a *App) Stop() {
	a.ticker.Stop()
}

func (a *App) syncStores() {
	for range a.ticker.C {
		wg := sync.WaitGroup{}
		go a.runManagerSync(wg, a.UserStore.Manager)
		go a.runManagerSync(wg, a.SessionStore.Manager)
		go a.runManagerSync(wg, a.ProblemStore.Manager)
		go a.runManagerSync(wg, a.PermissionStore.Manager)
		go a.runManagerSync(wg, a.RoleStore.Manager)
		wg.Wait()
	}
}

func (a *App) runManagerSync(wg sync.WaitGroup, m *models.ChangeManager) {
	wg.Add(1)
	defer wg.Done()
	if err := m.Sync(); err != nil {
		log.Print("unable to sync store: ", err)
	}
}
