package db

import (
	"context"
	"fmt"

	"github.com/udovin/gosql"
)

// Object represents an object from store.
type ObjectPtr[T any] interface {
	*T
	// ObjectID should return sequential ID of object.
	ObjectID() int64
	// SetObjectID should set sequential ID of object.
	SetObjectID(int64)
}

type FindObjectsOption interface {
	UpdateSelect(query gosql.SelectQuery)
}

// ObjectROStore represents read-only store for objects.
type ObjectROStore[T any] interface {
	// LoadObjects should load objects from store.
	LoadObjects(ctx context.Context) (Rows[T], error)
	// FindObjects should bind objects with specified expression.
	FindObjects(
		ctx context.Context, options ...FindObjectsOption,
	) (Rows[T], error)
}

// ObjectStore represents persistent store for objects.
type ObjectStore[T any, TPtr ObjectPtr[T]] interface {
	ObjectROStore[T]
	// CreateObject should create a new object set valid ID.
	CreateObject(ctx context.Context, object TPtr) error
	// UpdateObject should update object with specified ID.
	UpdateObject(ctx context.Context, object TPtr) error
	// DeleteObject should delete existing object from the store.
	DeleteObject(ctx context.Context, id int64) error
}

type objectStore[T any, TPtr ObjectPtr[T]] struct {
	db      *gosql.DB
	id      string
	table   string
	columns []string
}

func (s *objectStore[T, TPtr]) LoadObjects(ctx context.Context) (Rows[T], error) {
	builder := s.db.Select(s.table)
	builder.SetNames(s.columns...)
	builder.SetOrderBy(gosql.Ascending(s.id))
	rows, err := GetRunner(ctx, s.db.RO).QueryContext(ctx, builder.String())
	if err != nil {
		return nil, err
	}
	if err := checkColumns(rows, s.columns); err != nil {
		return nil, fmt.Errorf("store %q: %w", s.table, err)
	}
	return newRowReader[T](rows), nil
}

func (s *objectStore[T, TPtr]) FindObjects(
	ctx context.Context, options ...FindObjectsOption,
) (Rows[T], error) {
	builder := s.db.Select(s.table)
	builder.SetNames(s.columns...)
	builder.SetOrderBy(gosql.Ascending(s.id))
	for _, option := range options {
		option.UpdateSelect(builder)
	}
	query, values := builder.Build()
	rows, err := GetRunner(ctx, s.db.RO).QueryContext(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	if err := checkColumns(rows, s.columns); err != nil {
		return nil, fmt.Errorf("store %q: %w", s.table, err)
	}
	return newRowReader[T](rows), nil
}

func (s *objectStore[T, TPtr]) CreateObject(ctx context.Context, object TPtr) error {
	var id int64
	if err := insertRow(ctx, s.db, *object, &id, s.id, s.table); err != nil {
		return err
	}
	object.SetObjectID(id)
	return nil
}

func (s *objectStore[T, TPtr]) UpdateObject(ctx context.Context, object TPtr) error {
	return updateRow(ctx, s.db, *object, object.ObjectID(), s.id, s.table)
}

func (s *objectStore[T, TPtr]) DeleteObject(ctx context.Context, id int64) error {
	return deleteRow(ctx, s.db, id, s.id, s.table)
}

// NewObjectStore creates a new store for objects of specified type.
func NewObjectStore[T any, TPtr ObjectPtr[T]](
	id, table string, db *gosql.DB,
) ObjectStore[T, TPtr] {
	return &objectStore[T, TPtr]{
		db:      db,
		id:      id,
		table:   table,
		columns: getColumns[T](),
	}
}

type FindQuery struct {
	Where   gosql.BoolExpression
	Limit   int
	OrderBy []any
}

func (q FindQuery) UpdateSelect(query gosql.SelectQuery) {
	query.SetWhere(q.Where)
	query.SetLimit(q.Limit)
	if q.OrderBy != nil {
		query.SetOrderBy(q.OrderBy...)
	}
}

type WithLimit int

func (q WithLimit) UpdateSelect(query gosql.SelectQuery) {
	query.SetLimit(int(q))
}
