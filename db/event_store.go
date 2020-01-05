package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"
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

// EventROStore represents read-only store for events.
type EventROStore interface {
	// LoadEvents should load events from store in specified range.
	LoadEvents(tx *sql.Tx, begin, end int64) (EventReader, error)
}

// EventStore represents persistent store for events.
type EventStore interface {
	EventROStore
	// CreateEvent should create a new event and return copy
	// that has correct EventID.
	CreateEvent(tx *sql.Tx, event Event) (Event, error)
}

type eventStore struct {
	typ     reflect.Type
	id      string
	table   string
	dialect Dialect
}

func (s *eventStore) LoadEvents(
	tx *sql.Tx, begin, end int64,
) (EventReader, error) {
	rows, err := tx.Query(
		fmt.Sprintf(
			"SELECT %s FROM %q WHERE %q >= $1 AND %q < $2 ORDER BY %q",
			prepareSelect(s.typ), s.table, s.id, s.id, s.id,
		),
		begin, end,
	)
	if err != nil {
		return nil, err
	}
	return &eventReader{typ: s.typ, rows: rows}, nil
}

func (s *eventStore) CreateEvent(tx *sql.Tx, event Event) (Event, error) {
	row, err := insertRow(tx, event, s.id, s.table, s.dialect)
	if err != nil {
		return nil, err
	}
	return row.(Event), nil
}

// NewEventStore creates a new store for events of specified type.
func NewEventStore(event Event, id, table string, dialect Dialect) EventStore {
	return &eventStore{
		typ:     reflect.TypeOf(event),
		id:      id,
		table:   table,
		dialect: dialect,
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
	var v interface{}
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
