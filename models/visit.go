package models

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/udovin/gosql"
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

// VisitStore represents visit store.
type VisitStore struct {
	db     *gosql.DB
	events db.EventStore
}

// MakeFromContext creates Visit from context.
func (s *VisitStore) MakeFromContext(c echo.Context) Visit {
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

// CreateTx creates a new visit in the events.
func (s *VisitStore) CreateTx(tx gosql.WeakTx, visit Visit) (Visit, error) {
	event, err := s.events.CreateEvent(tx, visit)
	if err != nil {
		return Visit{}, err
	}
	return event.(Visit), nil
}

// NewVisitStore creates a new instance of ViewStore.
func NewVisitStore(dbConn *gosql.DB, table string) *VisitStore {
	return &VisitStore{
		db:     dbConn,
		events: db.NewEventStore(Visit{}, "id", table, dbConn.Dialect()),
	}
}
