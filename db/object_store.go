package db

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/udovin/gosql"
)

// Object represents an object from store.
type Object interface {
	// ObjectID should return sequential ID of object.
	ObjectID() int64
}

// ObjectReader represents reader for objects.
type ObjectReader interface {
	// Next should read next object and return true if object exists.
	Next() bool
	// Object should return current object.
	Object() Object
	// Close should close reader.
	Close() error
	// Err should return error that occurred during reading.
	Err() error
}

// ObjectROStore represents read-only store for objects.
type ObjectROStore interface {
	// LoadObjects should load objects from store.
	LoadObjects(tx gosql.WeakTx) (ObjectReader, error)
	// FindObjects should bind objects with specified expression.
	FindObjects(
		tx gosql.WeakTx, where string, args ...interface{},
	) (ObjectReader, error)
}

// ObjectStore represents persistent store for objects.
type ObjectStore interface {
	ObjectROStore
	// CreateObject should create a new object and return copy
	// that has correct ObjectID.
	CreateObject(tx gosql.WeakTx, object Object) (Object, error)
	// UpdateObject should update object with specified ObjectID and
	// return copy with updated fields.
	UpdateObject(tx gosql.WeakTx, object Object) (Object, error)
	// DeleteObject should delete existing object from the store.
	DeleteObject(tx gosql.WeakTx, id int64) error
}

type objectStore struct {
	typ     reflect.Type
	id      string
	table   string
	dialect gosql.Dialect
}

func (s *objectStore) LoadObjects(tx gosql.WeakTx) (ObjectReader, error) {
	rows, err := tx.Query(
		fmt.Sprintf(
			"SELECT %s FROM %q ORDER BY %q",
			prepareSelect(s.typ), s.table, s.id,
		),
	)
	if err != nil {
		return nil, err
	}
	if err := checkColumns(s.typ, rows); err != nil {
		return nil, err
	}
	return &objectReader{typ: s.typ, rows: rows}, nil
}

func (s *objectStore) FindObjects(
	tx gosql.WeakTx, where string, args ...interface{},
) (ObjectReader, error) {
	rows, err := tx.Query(
		fmt.Sprintf(
			"SELECT %s FROM %q WHERE %s",
			prepareSelect(s.typ), s.table, where,
		),
		args...,
	)
	if err != nil {
		return nil, err
	}
	if err := checkColumns(s.typ, rows); err != nil {
		return nil, err
	}
	return &objectReader{typ: s.typ, rows: rows}, nil
}

func (s *objectStore) CreateObject(tx gosql.WeakTx, object Object) (Object, error) {
	typ := reflect.TypeOf(object)
	if typ.Name() != s.typ.Name() || typ.PkgPath() != s.typ.PkgPath() {
		return nil, fmt.Errorf("expected %v type", s.typ)
	}
	row, err := insertRow(tx, object, s.id, s.table, s.dialect)
	if err != nil {
		return nil, err
	}
	return row.(Object), nil
}

func (s *objectStore) UpdateObject(tx gosql.WeakTx, object Object) (Object, error) {
	typ := reflect.TypeOf(object)
	if typ.Name() != s.typ.Name() || typ.PkgPath() != s.typ.PkgPath() {
		return nil, fmt.Errorf("expected %v type", s.typ)
	}
	row, err := updateRow(tx, object, s.id, s.table)
	if err != nil {
		return nil, err
	}
	return row.(Object), nil
}

func (s *objectStore) DeleteObject(tx gosql.WeakTx, id int64) error {
	return deleteRow(tx, id, s.id, s.table)
}

// NewObjectStore creates a new store for objects of specified type.
func NewObjectStore(
	object Object, id, table string, dialect gosql.Dialect,
) ObjectStore {
	return &objectStore{
		typ:     reflect.TypeOf(object),
		id:      id,
		table:   table,
		dialect: dialect,
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
	v, r.err = scanRow(r.typ, r.rows)
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
