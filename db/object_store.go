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
type ObjectReader[T Object] interface {
	// Next should read next object and return true if object exists.
	Next() bool
	// Object should return current object.
	Object() T
	// Close should close reader.
	Close() error
	// Err should return error that occurred during reading.
	Err() error
}

// ObjectROStore represents read-only store for objects.
type ObjectROStore[T Object] interface {
	// LoadObjects should load objects from store.
	LoadObjects(tx gosql.WeakTx) (ObjectReader[T], error)
	// FindObjects should bind objects with specified expression.
	FindObjects(
		tx gosql.WeakTx, where string, args ...any,
	) (ObjectReader[T], error)
}

// ObjectStore represents persistent store for objects.
type ObjectStore[T Object] interface {
	ObjectROStore[T]
	// CreateObject should create a new object and return copy
	// that has correct ObjectID.
	CreateObject(tx gosql.WeakTx, object *T) error
	// UpdateObject should update object with specified ObjectID and
	// return copy with updated fields.
	UpdateObject(tx gosql.WeakTx, object *T) error
	// DeleteObject should delete existing object from the store.
	DeleteObject(tx gosql.WeakTx, id int64) error
}

type objectStore[T Object] struct {
	typ     reflect.Type
	id      string
	table   string
	dialect gosql.Dialect
}

func (s *objectStore[T]) LoadObjects(tx gosql.WeakTx) (ObjectReader[T], error) {
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
	return &objectReader[T]{typ: s.typ, rows: rows}, nil
}

func (s *objectStore[T]) FindObjects(
	tx gosql.WeakTx, where string, args ...any,
) (ObjectReader[T], error) {
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
	return &objectReader[T]{typ: s.typ, rows: rows}, nil
}

func (s *objectStore[T]) CreateObject(tx gosql.WeakTx, object *T) error {
	row, err := insertRow(tx, *object, s.id, s.table, s.dialect)
	if err != nil {
		return err
	}
	*object = row.(T)
	return nil
}

func (s *objectStore[T]) UpdateObject(tx gosql.WeakTx, object *T) error {
	row, err := updateRow(tx, *object, s.id, s.table)
	if err != nil {
		return err
	}
	*object = row.(T)
	return nil
}

func (s *objectStore[T]) DeleteObject(tx gosql.WeakTx, id int64) error {
	return deleteRow(tx, id, s.id, s.table)
}

// NewObjectStore creates a new store for objects of specified type.
func NewObjectStore[T Object](
	id, table string, dialect gosql.Dialect,
) ObjectStore[T] {
	var object T
	return &objectStore[T]{
		typ:     reflect.TypeOf(object),
		id:      id,
		table:   table,
		dialect: dialect,
	}
}

type objectReader[T Object] struct {
	typ    reflect.Type
	rows   *sql.Rows
	err    error
	object T
}

func (r *objectReader[T]) Next() bool {
	if !r.rows.Next() {
		return false
	}
	var v any
	v, r.err = scanRow(r.typ, r.rows)
	if r.err == nil {
		r.object = v.(T)
	}
	return r.err == nil
}

func (r *objectReader[T]) Object() T {
	return r.object
}

func (r *objectReader[T]) Close() error {
	return r.rows.Close()
}

func (r *objectReader[T]) Err() error {
	if err := r.rows.Err(); err != nil {
		return err
	}
	return r.err
}
