package db

import (
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
type EventReader interface {
	// Next should read next event and return true if event exists.
	Next() bool
	// Event should return current event.
	Event() Event
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
type EventROStore interface {
	// LastEventID should return last event ID or sql.ErrNoRows
	// if there is no events.
	LastEventID(tx gosql.WeakTx) (int64, error)
	// LoadEvents should load events from store in specified range.
	LoadEvents(tx gosql.WeakTx, ranges []EventRange) (EventReader, error)
}

// EventStore represents persistent store for events.
type EventStore interface {
	EventROStore
	// CreateEvent should create a new event and return copy
	// that has correct EventID.
	CreateEvent(tx gosql.WeakTx, event Event) (Event, error)
}

type eventStore struct {
	typ   reflect.Type
	db    *gosql.DB
	id    string
	table string
}

// LastEventID returns last event ID or sql.ErrNoRows
// if there is no events.
func (s *eventStore) LastEventID(tx gosql.WeakTx) (int64, error) {
	row := tx.QueryRow(
		fmt.Sprintf("SELECT max(%q) FROM %q", s.id, s.table),
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

func (s *eventStore) getEventsWhere(ranges []EventRange) gosql.BoolExpression {
	if len(ranges) == 0 {
		return nil
	}
	where := ranges[0].getWhere(s.id)
	for _, rng := range ranges[1:] {
		where = where.Or(rng.getWhere(s.id))
	}
	return where
}

func (s *eventStore) LoadEvents(
	tx gosql.WeakTx, ranges []EventRange,
) (EventReader, error) {
	query, values := s.db.Select(s.table).
		Names(prepareNames(s.typ)...).
		Where(s.getEventsWhere(ranges)).
		OrderBy(gosql.Ascending(s.id)).
		// Limit(1000).
		Build()
	rows, err := tx.Query(query, values...)
	if err != nil {
		return nil, err
	}
	if err := checkColumns(s.typ, rows); err != nil {
		return nil, err
	}
	return &eventReader{typ: s.typ, rows: rows}, nil
}

func (s *eventStore) CreateEvent(tx gosql.WeakTx, event Event) (Event, error) {
	typ := reflect.TypeOf(event)
	if typ.Name() != s.typ.Name() || typ.PkgPath() != s.typ.PkgPath() {
		return nil, fmt.Errorf("expected %v type", s.typ)
	}
	row, err := insertRow(tx, event, s.id, s.table, s.db.Dialect())
	if err != nil {
		return nil, err
	}
	return row.(Event), nil
}

// NewEventStore creates a new store for events of specified type.
func NewEventStore(event Event, id, table string, db *gosql.DB) EventStore {
	return &eventStore{
		typ:   reflect.TypeOf(event),
		db:    db,
		id:    id,
		table: table,
	}
}

type eventReader struct {
	typ   reflect.Type
	rows  *sql.Rows
	err   error
	event Event
}

func (r *eventReader) Next() bool {
	if !r.rows.Next() {
		return false
	}
	var v any
	v, r.err = scanRow(r.typ, r.rows)
	if r.err == nil {
		r.event = v.(Event)
	}
	return r.err == nil
}

func (r *eventReader) Event() Event {
	return r.event
}

func (r *eventReader) Close() error {
	return r.rows.Close()
}

func (r *eventReader) Err() error {
	if err := r.rows.Err(); err != nil {
		return err
	}
	return r.err
}
