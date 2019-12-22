package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"
)

// Event represents an event from store
type Event interface {
	// EventID should return sequential ID of event
	EventID() int64
	// EventTime should return time when event occurred
	EventTime() time.Time
}

// EventReader represents reader for events
type EventReader interface {
	// Next should read next change and return true if change exists
	Next() bool
	// Event should return current change
	Event() Event
	// Close should close reader
	Close() error
	// Err should return error that occurred during reading
	Err() error
}

// EventROStore represents persistent store for event
type EventROStore interface {
	// LoadEvents should load events from store in specified range
	LoadEvents(tx *sql.Tx, begin, end int64) (EventReader, error)
}

type EventStore interface {
	EventROStore
	// CreateEvent should create a new event
	CreateEvent(tx *sql.Tx, event Event) (Event, error)
}

type eventStore struct {
	typ   reflect.Type
	table string
	id    string
}

type eventReader struct {
	typ   reflect.Type
	rows  *sql.Rows
	err   error
	event Event
}

func wrapStructScan(v interface{}) (fields []interface{}) {
	var wrap func(reflect.Value)
	wrap = func(v reflect.Value) {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			if _, ok := t.Field(i).Tag.Lookup("db"); ok {
				fields = append(fields, v.Field(i).Addr().Interface())
			}
			if t.Field(i).Anonymous {
				wrap(v.Field(i))
			}
		}
	}
	wrap(reflect.ValueOf(v).Elem())
	return fields
}

func (r *eventReader) Next() bool {
	if !r.rows.Next() {
		return false
	}
	r.event = reflect.New(r.typ).Interface().(Event)
	r.err = r.rows.Scan(wrapStructScan(&r.event)...)
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

func (s *eventStore) LoadEvents(
	tx *sql.Tx, begin, end int64,
) (EventReader, error) {
	rows, err := tx.Query(
		fmt.Sprintf(
			"SELECT %s FROM %q WHERE %q >= $1 AND %q < $2",
			"", s.table, s.id, s.id,
		),
		begin, end,
	)
	if err != nil {
		return nil, err
	}
	return &eventReader{typ: s.typ, rows: rows}, nil
}

func (s *eventStore) CreateEvent(
	tx *sql.Tx, event Event,
) (Event, error) {
	return nil, nil
}

// NewEventStore creates a new store for events of specified type
func NewEventStore(event Event, table string, id string) EventStore {
	return &eventStore{
		typ:   reflect.TypeOf(event),
		table: table,
		id:    id,
	}
}
