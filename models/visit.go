package models

import (
	"database/sql"
	"time"

	"github.com/labstack/echo"

	"github.com/udovin/solve/db"
)

// Visit represents user visit.
type Visit struct {
	ID         int64  `db:"id"`
	Time       int64  `db:"time"`
	AccountID  NInt64 `db:"account_id"`
	SessionID  NInt64 `db:"session_id"`
	Host       string `db:"host"`
	Protocol   string `db:"protocol"`
	Method     string `db:"method" `
	RemoteAddr string `db:"remote_addr"`
	UserAgent  string `db:"user_agent"`
	Path       string `db:"path"`
	RealIP     string `db:"real_ip"`
	Status     int    `db:"status"`
}

// EventID returns ID of visit.
func (o Visit) EventID() int64 {
	return o.ID
}

// EventTime return time of visit.
func (o Visit) EventTime() time.Time {
	return time.Unix(o.Time, 0)
}

// VisitManager represents visit manager.
type VisitManager struct {
	store db.EventStore
}

// MakeFromContext creates Visit from context.
func (m *VisitManager) MakeFromContext(c echo.Context) Visit {
	return Visit{
		Time:       time.Now().Unix(),
		Host:       c.Request().Host,
		Protocol:   c.Request().Proto,
		Method:     c.Request().Method,
		RemoteAddr: c.Request().RemoteAddr,
		UserAgent:  c.Request().UserAgent(),
		Path:       c.Request().URL.RequestURI(),
		RealIP:     c.RealIP(),
	}
}

// CreateTx creates a new visit in the store.
func (m *VisitManager) CreateTx(tx *sql.Tx, visit Visit) (Visit, error) {
	event, err := m.store.CreateEvent(tx, visit)
	if err != nil {
		return Visit{}, err
	}
	return event.(Visit), nil
}

// NewVisitManager creates a new instance of ViewManager.
func NewVisitManager(table string, dialect db.Dialect) *VisitManager {
	return &VisitManager{
		store: db.NewEventStore(Visit{}, "id", table, dialect),
	}
}
