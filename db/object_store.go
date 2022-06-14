package db

import (
	"context"

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
	LoadObjects(ctx context.Context) (RowReader[T], error)
	// FindObjects should bind objects with specified expression.
	FindObjects(
		ctx context.Context, where gosql.BoolExpression,
	) (RowReader[T], error)
}

// ObjectStore represents persistent store for objects.
type ObjectStore[T Object] interface {
	ObjectROStore[T]
	// CreateObject should create a new object set valid ID.
	CreateObject(ctx context.Context, object *T) error
	// UpdateObject should update object with specified ID.
	UpdateObject(ctx context.Context, object *T) error
	// DeleteObject should delete existing object from the store.
	DeleteObject(ctx context.Context, id int64) error
}

type objectStore[T Object] struct {
	db      *gosql.DB
	id      string
	table   string
	columns []string
}

func (s *objectStore[T]) LoadObjects(ctx context.Context) (RowReader[T], error) {
	builder := s.db.Select(s.table)
	builder.SetNames(s.columns...)
	builder.SetOrderBy(gosql.Ascending(s.id))
	rows, err := GetRunner(ctx, s.db).QueryContext(ctx, builder.String())
	if err != nil {
		return nil, err
	}
	if err := checkColumns(rows, s.columns); err != nil {
		return nil, err
	}
	return newRowReader[T](rows), nil
}

func (s *objectStore[T]) FindObjects(
	ctx context.Context, where gosql.BoolExpression,
) (RowReader[T], error) {
	builder := s.db.Select(s.table)
	builder.SetNames(s.columns...)
	builder.SetWhere(where)
	builder.SetOrderBy(gosql.Ascending(s.id))
	query, values := builder.Build()
	rows, err := GetRunner(ctx, s.db).QueryContext(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	if err := checkColumns(rows, s.columns); err != nil {
		return nil, err
	}
	return newRowReader[T](rows), nil
}

func (s *objectStore[T]) CreateObject(ctx context.Context, object *T) error {
	return insertRow(ctx, s.db, object, s.id, s.table)
}

func (s *objectStore[T]) UpdateObject(ctx context.Context, object *T) error {
	return updateRow(ctx, s.db, object, s.id, s.table)
}

func (s *objectStore[T]) DeleteObject(ctx context.Context, id int64) error {
	return deleteRow(ctx, s.db, id, s.id, s.table)
}

// NewObjectStore creates a new store for objects of specified type.
func NewObjectStore[T Object](
	id, table string, db *gosql.DB,
) ObjectStore[T] {
	return &objectStore[T]{
		db:      db,
		id:      id,
		table:   table,
		columns: prepareNames[T](),
	}
}
