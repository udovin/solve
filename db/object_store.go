package db

import (
	"database/sql"
	"fmt"
	"reflect"
)

// Object represents an object from store
type Object interface {
	// ObjectID should return sequential ID of object
	ObjectID() int64
}

// ObjectReader represents reader for objects
type ObjectReader interface {
	// Next should read next object and return true if object exists
	Next() bool
	// Object should return current object
	Object() Object
	// Close should close reader
	Close() error
	// Err should return error that occurred during reading
	Err() error
}

// ObjectROStore represents read-only store for objects
type ObjectROStore interface {
	// LoadObjects should load objects from store
	LoadObjects(tx *sql.Tx) (ObjectReader, error)
}

// ObjectStore represents persistent store for objects
type ObjectStore interface {
	ObjectROStore
	// CreateObject should create a new object and return copy
	// that has correct ObjectID
	CreateObject(tx *sql.Tx, object Object) (Object, error)
}

type objectStore struct {
	typ   reflect.Type
	table string
	id    string
	dbms  DBMS
}

func (s *objectStore) LoadObjects(tx *sql.Tx) (ObjectReader, error) {
	rows, err := tx.Query(
		fmt.Sprintf(
			"SELECT %s FROM %q ORDER BY %q",
			selectValue(s.typ), s.table, s.id,
		),
	)
	if err != nil {
		return nil, err
	}
	return &objectReader{typ: s.typ, rows: rows}, nil
}

func (s *objectStore) CreateObject(tx *sql.Tx, object Object) (Object, error) {
	value := cloneValue(object)
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
	return value.Interface().(Object), nil
}

// NewObjectStore creates a new store for objects of specified type
func NewObjectStore(
	object Object, table string, id string, dbms DBMS,
) ObjectStore {
	return &objectStore{
		typ:   reflect.TypeOf(object),
		table: table,
		id:    id,
		dbms:  dbms,
	}
}

type objectReader struct {
	typ    reflect.Type
	rows   *sql.Rows
	err    error
	object Object
}

func (r *objectReader) Next() bool {
	if !r.rows.Next() {
		return false
	}
	var v interface{}
	v, r.err = scanValue(r.typ, r.rows)
	if r.err == nil {
		r.object = v.(Object)
	}
	return r.err == nil
}

func (r *objectReader) Object() Object {
	return r.object
}

func (r *objectReader) Close() error {
	return r.rows.Close()
}

func (r *objectReader) Err() error {
	if err := r.rows.Err(); err != nil {
		return err
	}
	return r.err
}
