package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
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
	// CreateEvent should create a new event and return event copy
	// that has correct EventID
	//
	// There is no guarantee that returned event has the same type
	// but should be guaranteed that EventID() is correct. So if you
	// do not know anything about store, you should assume that only
	// EventID() has correct value.
	CreateEvent(tx *sql.Tx, event Event) (Event, error)
}

type eventStore struct {
	typ   reflect.Type
	table string
	id    string
	dbms  DBMS
}

func (s *eventStore) LoadEvents(
	tx *sql.Tx, begin, end int64,
) (EventReader, error) {
	rows, err := tx.Query(
		fmt.Sprintf(
			"SELECT %s FROM %q WHERE %q >= $1 AND %q < $2",
			selectStruct(s.typ), s.table, s.id, s.id,
		),
		begin, end,
	)
	if err != nil {
		return nil, err
	}
	return &eventReader{typ: s.typ, rows: rows}, nil
}

type stubEvent int64

func (e stubEvent) EventID() int64 {
	return int64(e)
}

func (e stubEvent) EventTime() time.Time {
	return time.Time{}
}

func (s *eventStore) CreateEvent(tx *sql.Tx, event Event) (Event, error) {
	cols, keys, vals := insertObject(event, s.id)
	var id int64
	switch s.dbms {
	case Postgres:
		rows := tx.QueryRow(
			fmt.Sprintf(
				"INSERT INTO %q (%s) VALUES (%s) RETURNING %q",
				s.table, cols, keys, s.id,
			),
			vals...,
		)
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
	default:
		res, err := tx.Exec(
			fmt.Sprintf(
				"INSERT INTO %q (%s) VALUES (%s)",
				s.table, cols, keys,
			),
			vals...,
		)
		if err != nil {
			return nil, err
		}
		if id, err = res.LastInsertId(); err != nil {
			return nil, err
		}
	}
	return stubEvent(id), nil
}

// NewEventStore creates a new store for events of specified type
func NewEventStore(
	event Event, table string, id string, dbms DBMS,
) EventStore {
	return &eventStore{
		typ:   reflect.TypeOf(event),
		table: table,
		id:    id,
		dbms:  dbms,
	}
}

func selectStruct(typ reflect.Type) string {
	var cols strings.Builder
	var recursive func(reflect.Type)
	recursive = func(t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			if db, ok := t.Field(i).Tag.Lookup("db"); ok {
				name := strings.Split(db, ",")[0]
				if cols.Len() > 0 {
					cols.WriteRune(',')
				}
				cols.WriteString(fmt.Sprintf("%q", name))
			} else if t.Field(i).Anonymous {
				recursive(t.Field(i).Type)
			}
		}
	}
	recursive(typ)
	return cols.String()
}

func insertObject(
	value interface{}, id string,
) (string, string, []interface{}) {
	var cols strings.Builder
	var keys strings.Builder
	var vals []interface{}
	var it int
	var recursive func(reflect.Value)
	recursive = func(v reflect.Value) {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			if db, ok := t.Field(i).Tag.Lookup("db"); ok {
				name := strings.Split(db, ",")[0]
				if name == id {
					continue
				}
				if it > 0 {
					cols.WriteRune(',')
					keys.WriteRune(',')
				}
				it++
				cols.WriteString(fmt.Sprintf("%q", name))
				keys.WriteString(fmt.Sprintf("$%d", it))
				vals = append(vals, v.Field(i).Interface())
			} else if t.Field(i).Anonymous {
				recursive(v.Field(i))
			}
		}
	}
	recursive(reflect.ValueOf(value))
	return cols.String(), keys.String(), vals
}

func scanObject(typ reflect.Type, rows *sql.Rows) (interface{}, error) {
	value := reflect.New(typ).Elem()
	var fields []interface{}
	var recursive func(reflect.Value)
	recursive = func(v reflect.Value) {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			if _, ok := t.Field(i).Tag.Lookup("db"); ok {
				fields = append(fields, v.Field(i).Addr().Interface())
			} else if t.Field(i).Anonymous {
				recursive(v.Field(i))
			}
		}
	}
	recursive(value)
	err := rows.Scan(fields...)
	return value.Interface(), err
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
	v, r.err = scanObject(r.typ, r.rows)
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
