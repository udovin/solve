package db

import (
	"database/sql"
	"fmt"
	"reflect"
)

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

func NewEventStore(event Event, table string, id string) EventStore {
	return &eventStore{
		typ:   reflect.TypeOf(event),
		table: table,
		id:    id,
	}
}
