// Package db provides implementation of generic object and event stores.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/udovin/gosql"
)

type dbKey struct{}

func WithRunner(ctx context.Context, db gosql.Runner) context.Context {
	return context.WithValue(ctx, dbKey{}, db)
}

func GetRunner(ctx context.Context, db gosql.Runner) gosql.Runner {
	if r, ok := ctx.Value(dbKey{}).(gosql.Runner); ok {
		return r
	}
	return db
}

func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return WithRunner(ctx, tx)
}

func GetTx(ctx context.Context) *sql.Tx {
	if tx, ok := ctx.Value(dbKey{}).(*sql.Tx); ok {
		return tx
	}
	return nil
}

// Rows represents reader for events.
type Rows[T any] interface {
	// Next should read next event and return true if event exists.
	Next() bool
	// Row should return current row.
	Row() T
	// Close should close reader.
	Close() error
	// Err should return error that occurred during reading.
	Err() error
}

type rowReader[T any] struct {
	rows *sql.Rows
	err  error
	row  T
	// refs contains pointers for each field in row.
	refs []any
}

func (r *rowReader[T]) Next() bool {
	if !r.rows.Next() {
		return false
	}
	r.err = r.rows.Scan(r.refs...)
	return r.err == nil
}

func (r *rowReader[T]) Row() T {
	return r.row
}

func (r *rowReader[T]) Close() error {
	return r.rows.Close()
}

func (r *rowReader[T]) Err() error {
	if err := r.rows.Err(); err != nil {
		return err
	}
	return r.err
}

func getRowFields[T any](row *T) []any {
	var fields []any
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
	recursive(reflect.ValueOf(row).Elem())
	return fields
}

func newRowReader[T any](rows *sql.Rows) *rowReader[T] {
	r := &rowReader[T]{rows: rows}
	r.refs = getRowFields(&r.row)
	return r
}

type sliceRows[T any] struct {
	rows []T
	pos  int
}

func (r *sliceRows[T]) Next() bool {
	r.pos++
	return r.pos < len(r.rows)
}

func (r *sliceRows[T]) Row() T {
	return r.rows[r.pos]
}

func (r *sliceRows[T]) Close() error {
	return nil
}

func (r *sliceRows[T]) Err() error {
	return nil
}

func NewSliceRows[T any](rows []T) Rows[T] {
	return &sliceRows[T]{rows: rows, pos: -1}
}

func checkColumns(rows *sql.Rows, cols []string) error {
	rowCols, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(cols) != len(rowCols) {
		return fmt.Errorf("result has invalid column sequence: %v != %v", cols, rowCols)
	}
	for i := 0; i < len(cols); i++ {
		if cols[i] != rowCols[i] {
			return fmt.Errorf("result has invalid column sequence: %v != %v", cols, rowCols)
		}
	}
	return nil
}

func getColumns[T any]() []string {
	var cols []string
	var recursive func(reflect.Type)
	recursive = func(t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			if db, ok := t.Field(i).Tag.Lookup("db"); ok {
				name := strings.Split(db, ",")[0]
				cols = append(cols, name)
			} else if t.Field(i).Anonymous {
				recursive(t.Field(i).Type)
			}
		}
	}
	var object T
	recursive(reflect.TypeOf(object))
	return cols
}

func prepareUpsert(value reflect.Value, id string) ([]string, []any) {
	var cols []string
	var vals []any
	var recursive func(reflect.Value)
	recursive = func(v reflect.Value) {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			if db, ok := t.Field(i).Tag.Lookup("db"); ok {
				name := strings.Split(db, ",")[0]
				if name == id {
					continue
				}
				cols = append(cols, name)
				vals = append(vals, v.Field(i).Interface())
			} else if t.Field(i).Anonymous {
				recursive(v.Field(i))
			}
		}
	}
	recursive(value)
	return cols, vals
}

func insertRow[T any](
	ctx context.Context, db *gosql.DB, row T, rowID *int64,
	id, table string,
) error {
	cols, vals := prepareUpsert(reflect.ValueOf(row), id)
	builder := db.Insert(table)
	builder.SetNames(cols...)
	builder.SetValues(vals...)
	switch b := builder.(type) {
	case *gosql.PostgresInsertQuery:
		b.SetReturning(id)
		res := GetRunner(ctx, db).QueryRowContext(ctx, db.BuildString(builder), vals...)
		return res.Scan(rowID)
	default:
		res, err := GetRunner(ctx, db).ExecContext(ctx, db.BuildString(builder), vals...)
		if err != nil {
			return err
		}
		count, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("invalid amount of affected rows: %d", count)
		}
		*rowID, err = res.LastInsertId()
		return err
	}
}

func updateRow[T any](
	ctx context.Context, db *gosql.DB, row T, rowID int64,
	id, table string,
) error {
	cols, vals := prepareUpsert(reflect.ValueOf(row), id)
	builder := db.Update(table)
	builder.SetNames(cols...)
	builder.SetValues(vals...)
	builder.SetWhere(gosql.Column(id).Equal(rowID))
	query, values := db.Build(builder)
	res, err := GetRunner(ctx, db).ExecContext(ctx, query, values...)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count < 1 {
		return sql.ErrNoRows
	} else if count > 1 {
		return fmt.Errorf("updated %d objects", count)
	}
	return nil
}

func deleteRow(
	ctx context.Context, db *gosql.DB, rowID int64,
	id, table string,
) error {
	builder := db.Delete(table)
	builder.SetWhere(gosql.Column(id).Equal(rowID))
	query, values := db.Build(builder)
	res, err := GetRunner(ctx, db).ExecContext(ctx, query, values...)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count < 1 {
		return sql.ErrNoRows
	} else if count > 1 {
		return fmt.Errorf("deleted %d objects", count)
	}
	return nil
}
