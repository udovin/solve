package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
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

// EventReader represents reader for events.
type EventReader[T Event] interface {
	// Next should read next event and return true if event exists.
	Next() bool
	// Event should return current event.
	Event() T
	// Close should close reader.
	Close() error
	// Err should return error that occurred during reading.
	Err() error
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
	LoadEvents(ctx context.Context, ranges []EventRange) (EventReader[T], error)
}

// EventStore represents persistent store for events.
type EventStore[T Event] interface {
	EventROStore[T]
	// CreateEvent should create a new event and return copy
	// that has correct EventID.
	CreateEvent(ctx context.Context, event *T) error
}

type eventStore[T Event] struct {
	typ   reflect.Type
	db    *gosql.DB
	id    string
	table string
}

// LastEventID returns last event ID or sql.ErrNoRows
// if there is no events.
func (s *eventStore[T]) LastEventID(ctx context.Context) (int64, error) {
	row := s.db.QueryRowContext(
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
) (EventReader[T], error) {
	query, values := s.db.Select(s.table).
		Names(prepareNames(s.typ)...).
		Where(s.getEventsWhere(ranges)).
		OrderBy(gosql.Ascending(s.id)).
		// Limit(1000).
		Build()
	rows, err := s.db.QueryContext(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	if err := checkColumns(s.typ, rows); err != nil {
		return nil, err
	}
	return &eventReader[T]{typ: s.typ, rows: rows}, nil
}

func (s *eventStore[T]) CreateEvent(ctx context.Context, event *T) error {
	row, err := insertRow(getWeakTx(ctx, s.db), *event, s.id, s.table, s.db.Dialect())
	if err != nil {
		return err
	}
	*event = row.(T)
	return nil
}

func (s *eventStore[T]) CreateEventTx(tx gosql.WeakTx, event *T) error {
	return s.CreateEvent(wrapContext(tx), event)
}

// NewEventStore creates a new store for events of specified type.
func NewEventStore[T Event](id, table string, db *gosql.DB) EventStore[T] {
	var event T
	return &eventStore[T]{
		typ:   reflect.TypeOf(event),
		db:    db,
		id:    id,
		table: table,
	}
}

type eventReader[T Event] struct {
	typ   reflect.Type
	rows  *sql.Rows
	err   error
	event T
}

func (r *eventReader[T]) Next() bool {
	if !r.rows.Next() {
		return false
	}
	var v any
	v, r.err = scanRow(r.typ, r.rows)
	if r.err == nil {
		r.event = v.(T)
	}
	return r.err == nil
}

func (r *eventReader[T]) Event() T {
	return r.event
}

func (r *eventReader[T]) Close() error {
	return r.rows.Close()
}

func (r *eventReader[T]) Err() error {
	if err := r.rows.Err(); err != nil {
		return err
	}
	return r.err
}
