package api

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/logs"
)

func (v *View) StartDaemons() {
	v.visits = make(chan visitContext, 100)
	v.core.StartTask("visits", v.visitsDaemon)
	v.core.StartUniqueDaemon("session_cleanup", v.sessionCleanupDaemon)
	v.core.StartUniqueDaemon("token_cleanup", v.tokenCleanupDaemon)
}

type visitContext struct {
	Path   string
	Visit  models.Visit
	Logger echo.Logger
}

func (v *visitContext) Create(view *View) {
	if view.getBoolSetting("handlers."+v.Path+".log_visit", v.Logger).OrElse(true) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := view.core.Visits.Create(ctx, &v.Visit); err != nil {
			view.core.Logger().Error("Unable to create visit", err)
		}
	}
}

func (v *View) visitsDaemon(ctx context.Context) {
	for {
		select {
		case visit := <-v.visits:
			visit.Create(v)
		case <-ctx.Done():
			select {
			case visit := <-v.visits:
				visit.Create(v)
			default:
			}
			return
		}
	}
}

func (v *View) sessionCleanupDaemon(ctx context.Context) {
	cleanupTask := func() error {
		rows, err := v.core.Sessions.Find(ctx, db.FindQuery{
			Where: gosql.Column("expire_time").LessEqual(time.Now().Unix()),
		})
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			row := rows.Row()
			if err := v.core.Sessions.Delete(ctx, row.ID); err != nil {
				v.core.Logger().Warn(
					"Cannot remove expired session",
					logs.Any("id", row.ID),
					err,
				)
			}
			v.core.Logger().Info(
				"Removed expired session",
				logs.Any("id", row.ID),
				logs.Any("expire_time", row.ExpireTime),
			)
		}
		return rows.Err()
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	if err := cleanupTask(); err != nil {
		v.core.Logger().Warn("Sessions cleanup error", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := cleanupTask(); err != nil {
				v.core.Logger().Warn("Sessions cleanup error", err)
				return
			}
		}
	}
}

func (v *View) tokenCleanupDaemon(ctx context.Context) {
	cleanupTask := func() error {
		rows, err := v.core.Tokens.Find(ctx, db.FindQuery{
			Where: gosql.Column("expire_time").LessEqual(time.Now().Unix()),
		})
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			row := rows.Row()
			if err := v.core.Tokens.Delete(ctx, row.ID); err != nil {
				v.core.Logger().Warn(
					"Cannot remove expired token",
					logs.Any("id", row.ID),
					err,
				)
			}
			v.core.Logger().Info(
				"Removed expired token",
				logs.Any("id", row.ID),
				logs.Any("expire_time", row.ExpireTime),
			)
		}
		return rows.Err()
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	if err := cleanupTask(); err != nil {
		v.core.Logger().Warn("Tokens cleanup error", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := cleanupTask(); err != nil {
				v.core.Logger().Warn("Tokens cleanup error", err)
				return
			}
		}
	}
}
