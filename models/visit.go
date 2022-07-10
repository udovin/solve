package models

import (
	"context"
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

// SetEventID sets ID of visit.
func (o *Visit) SetEventID(id int64) {
	o.ID = id
}

// EventTime return time of visit.
func (o Visit) EventTime() time.Time {
	return time.Unix(o.Time, 0)
}

// VisitStore represents visit store.
type VisitStore struct {
	db     *gosql.DB
	events db.EventStore[Visit, *Visit]
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

// Create creates a new visit in the events.
func (s *VisitStore) Create(ctx context.Context, visit *Visit) error {
	return s.events.CreateEvent(ctx, visit)
}

// NewVisitStore creates a new instance of ViewStore.
func NewVisitStore(dbConn *gosql.DB, table string) *VisitStore {
	return &VisitStore{
		db:     dbConn,
		events: db.NewEventStore[Visit]("id", table, dbConn),
	}
}
