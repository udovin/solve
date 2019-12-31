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
	// Next should read next event and return true if event exists
	Next() bool
	// Event should return current event
	Event() Event
	// Close should close reader
	Close() error
	// Err should return error that occurred during reading
	Err() error
}

// EventROStore represents read-only store for events
type EventROStore interface {
	// LoadEvents should load events from store in specified range
	LoadEvents(tx *sql.Tx, begin, end int64) (EventReader, error)
}

// EventStore represents persistent store for events
type EventStore interface {
	EventROStore
	// CreateEvent should create a new event and return copy
	// that has correct EventID
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
			"SELECT %s FROM %q WHERE %q >= $1 AND %q < $2 ORDER BY %q",
			selectValue(s.typ), s.table, s.id, s.id, s.id,
		),
		begin, end,
	)
	if err != nil {
		return nil, err
	}
	return &eventReader{typ: s.typ, rows: rows}, nil
}

func (s *eventStore) CreateEvent(tx *sql.Tx, event Event) (Event, error) {
	value := cloneValue(event)
	cols, keys, vals, idPtr := insertValue(value, s.id)
	switch s.dbms {
	case Postgres:
		rows := tx.QueryRow(
			fmt.Sprintf(
				"INSERT INTO %q (%s) VALUES (%s) RETURNING %q",
				s.table, cols, keys, s.id,
			),
			vals...,
		)
		if err := rows.Scan(idPtr); err != nil {
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
		if *idPtr, err = res.LastInsertId(); err != nil {
			return nil, err
		}
	}
	return value.Interface().(Event), nil
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
	v, r.err = scanValue(r.typ, r.rows)
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

func selectValue(typ reflect.Type) string {
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

func cloneValue(value interface{}) reflect.Value {
	clone := reflect.New(reflect.TypeOf(value)).Elem()
	var recursive func(value, clone reflect.Value)
	recursive = func(value, clone reflect.Value) {
		t := value.Type()
		for i := 0; i < t.NumField(); i++ {
			if _, ok := t.Field(i).Tag.Lookup("db"); ok {
				clone.Field(i).Set(value.Field(i))
			} else if t.Field(i).Anonymous {
				recursive(value.Field(i), clone.Field(i))
			}
		}
	}
	recursive(reflect.ValueOf(value), clone)
	return clone
}

func insertValue(
	value reflect.Value, id string,
) (string, string, []interface{}, *int64) {
	var cols strings.Builder
	var keys strings.Builder
	var vals []interface{}
	var idPtr *int64
	var it int
	var recursive func(reflect.Value)
	recursive = func(v reflect.Value) {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			if db, ok := t.Field(i).Tag.Lookup("db"); ok {
				name := strings.Split(db, ",")[0]
				if name == id {
					idPtr = v.Field(i).Addr().Interface().(*int64)
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
	recursive(value)
	return cols.String(), keys.String(), vals, idPtr
}

func scanValue(typ reflect.Type, rows *sql.Rows) (interface{}, error) {
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
