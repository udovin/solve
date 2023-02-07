package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/udovin/gosql"
)

// EventPtr represents a mutable event from store.
type EventPtr[T any] interface {
	*T
	// EventID should return sequential ID of event.
	EventID() int64
	// SetEventID should set sequential ID of event.
	SetEventID(int64)
	// EventTime should return time when event occurred.
	EventTime() time.Time
}

// EventRange represents range [begin, end).
type EventRange struct {
	// Begin contains begin of range.
	//
	// Begin can not be greater than End and can not be less than 0.
	Begin int64
	// End contains end of range.
	//
	// If End == 0, then there is no upper limit.
	End int64
}

func (r EventRange) contains(id int64) bool {
	return id >= r.Begin && (r.End == 0 || id < r.End)
}

func (r EventRange) getWhere(name string) gosql.BoolExpression {
	column := gosql.Column(name)
	if r.End == 0 {
		return column.GreaterEqual(r.Begin)
	}
	if r.Begin+1 == r.End {
		return column.Equal(r.Begin)
	}
	return column.GreaterEqual(r.Begin).And(column.Less(r.End))
}

// EventROStore represents read-only store for events.
type EventROStore[T any] interface {
	// LastEventID should return last event ID or sql.ErrNoRows
	// if there is no events.
	LastEventID(ctx context.Context) (int64, error)
	// LoadEvents should load events from store in specified range.
	LoadEvents(ctx context.Context, ranges []EventRange) (Rows[T], error)
}

// EventStore represents persistent store for events.
type EventStore[T any, TPtr EventPtr[T]] interface {
	EventROStore[T]
	// CreateEvent should create a new event and return copy
	// that has correct ID.
	CreateEvent(ctx context.Context, event TPtr) error
}

type eventStore[T any, TPtr EventPtr[T]] struct {
	db      *gosql.DB
	id      string
	table   string
	columns []string
}

// LastEventID returns last event ID or sql.ErrNoRows
// if there is no events.
func (s *eventStore[T, TPtr]) LastEventID(ctx context.Context) (int64, error) {
	row := GetRunner(ctx, s.db.RO).QueryRowContext(
		ctx, fmt.Sprintf("SELECT max(%q) FROM %q", s.id, s.table),
	)
	var id *int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	if id == nil {
		return 0, sql.ErrNoRows
	}
	return *id, nil
}

func (s *eventStore[T, TPtr]) getEventsWhere(ranges []EventRange) gosql.BoolExpression {
	if len(ranges) == 0 {
		return nil
	}
	where := ranges[0].getWhere(s.id)
	for _, rng := range ranges[1:] {
		where = where.Or(rng.getWhere(s.id))
	}
	return where
}

func (s *eventStore[T, TPtr]) LoadEvents(
	ctx context.Context, ranges []EventRange,
) (Rows[T], error) {
	builder := s.db.Select(s.table)
	builder.SetNames(s.columns...)
	builder.SetWhere(s.getEventsWhere(ranges))
	builder.SetOrderBy(gosql.Ascending(s.id))
	query, values := builder.Build()
	rows, err := GetRunner(ctx, s.db.RO).QueryContext(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	if err := checkColumns(rows, s.columns); err != nil {
		return nil, fmt.Errorf("store %q: %w", s.table, err)
	}
	return newRowReader[T](rows), nil
}

func (s *eventStore[T, TPtr]) CreateEvent(ctx context.Context, event TPtr) error {
	var id int64
	if err := insertRow(ctx, s.db, *event, &id, s.id, s.table); err != nil {
		return err
	}
	event.SetEventID(id)
	return nil
}

// NewEventStore creates a new store for events of specified type.
func NewEventStore[T any, TPtr EventPtr[T]](id, table string, db *gosql.DB) EventStore[T, TPtr] {
	return &eventStore[T, TPtr]{
		db:      db,
		id:      id,
		table:   table,
		columns: getColumns[T](),
	}
}
