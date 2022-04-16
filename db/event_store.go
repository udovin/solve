package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/udovin/gosql"
)

// Event represents an event from store.
type Event interface {
	// EventID should return sequential ID of event.
	EventID() int64
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
type EventROStore[T Event] interface {
	// LastEventID should return last event ID or sql.ErrNoRows
	// if there is no events.
	LastEventID(ctx context.Context) (int64, error)
	// LoadEvents should load events from store in specified range.
	LoadEvents(ctx context.Context, ranges []EventRange) (RowReader[T], error)
}

// EventStore represents persistent store for events.
type EventStore[T Event] interface {
	EventROStore[T]
	// CreateEvent should create a new event and return copy
	// that has correct EventID.
	CreateEvent(ctx context.Context, event *T) error
}

type eventStore[T Event] struct {
	db      *gosql.DB
	id      string
	table   string
	columns []string
}

// LastEventID returns last event ID or sql.ErrNoRows
// if there is no events.
func (s *eventStore[T]) LastEventID(ctx context.Context) (int64, error) {
	tx := GetRunner(ctx, s.db)
	row := tx.QueryRowContext(
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

func (s *eventStore[T]) getEventsWhere(ranges []EventRange) gosql.BoolExpression {
	if len(ranges) == 0 {
		return nil
	}
	where := ranges[0].getWhere(s.id)
	for _, rng := range ranges[1:] {
		where = where.Or(rng.getWhere(s.id))
	}
	return where
}

func (s *eventStore[T]) LoadEvents(
	ctx context.Context, ranges []EventRange,
) (RowReader[T], error) {
	query, values := s.db.Select(s.table).
		Names(s.columns...).
		Where(s.getEventsWhere(ranges)).
		OrderBy(gosql.Ascending(s.id)).
		Build()
	tx := GetRunner(ctx, s.db)
	rows, err := tx.QueryContext(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	if err := checkColumns(rows, s.columns); err != nil {
		return nil, err
	}
	return &rowReader[T]{rows: rows}, nil
}

func (s *eventStore[T]) CreateEvent(ctx context.Context, event *T) error {
	return insertRow(ctx, s.db, event, s.id, s.table)
}

// NewEventStore creates a new store for events of specified type.
func NewEventStore[T Event](id, table string, db *gosql.DB) EventStore[T] {
	return &eventStore[T]{
		db:      db,
		id:      id,
		table:   table,
		columns: prepareNames[T](),
	}
}
