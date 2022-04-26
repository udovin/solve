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

type txKey struct{}

func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func GetTx(ctx context.Context) *sql.Tx {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	if ok {
		return tx
	}
	return nil
}

func GetRunner(ctx context.Context, db *gosql.DB) gosql.Runner {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return db
}

// RowReader represents reader for events.
type RowReader[T any] interface {
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

func checkColumns(rows *sql.Rows, cols []string) error {
	rowCols, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(cols) != len(rowCols) {
		return fmt.Errorf("result has invalid column sequence")
	}
	for i := 0; i < len(cols); i++ {
		if cols[i] != rowCols[i] {
			return fmt.Errorf("result has invalid column sequence")
		}
	}
	return nil
}

func prepareNames[T any]() []string {
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

func prepareInsert(value reflect.Value, id string) ([]string, []any, *int64) {
	var vals []any
	var idPtr *int64
	var cols []string
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
				cols = append(cols, name)
				vals = append(vals, v.Field(i).Interface())
			} else if t.Field(i).Anonymous {
				recursive(v.Field(i))
			}
		}
	}
	recursive(value)
	return cols, vals, idPtr
}

func insertRow[T any](
	ctx context.Context, db *gosql.DB, row *T,
	id, table string,
) error {
	cols, vals, idPtr := prepareInsert(reflect.ValueOf(row).Elem(), id)
	builder := db.Insert(table)
	builder.SetNames(cols...)
	builder.SetValues(vals...)
	switch b := builder.(type) {
	case *gosql.PostgresInsertQuery:
		b.SetReturning(id)
		rows := GetRunner(ctx, db).QueryRowContext(ctx, builder.String(), vals...)
		if err := rows.Scan(idPtr); err != nil {
			return err
		}
	default:
		res, err := GetRunner(ctx, db).ExecContext(ctx, builder.String(), vals...)
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
		if *idPtr, err = res.LastInsertId(); err != nil {
			return err
		}
	}
	return nil
}

func prepareUpdate(value reflect.Value, id string) ([]string, []any, int64) {
	var cols []string
	var vals []any
	var idValue int64
	var recursive func(reflect.Value)
	recursive = func(v reflect.Value) {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			if db, ok := t.Field(i).Tag.Lookup("db"); ok {
				name := strings.Split(db, ",")[0]
				if name == id {
					idValue = v.Field(i).Interface().(int64)
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
	return cols, vals, idValue
}

func updateRow[T any](
	ctx context.Context, db *gosql.DB, row *T,
	id, table string,
) error {
	cols, vals, idValue := prepareUpdate(reflect.ValueOf(row).Elem(), id)
	builder := db.Update(table)
	builder.SetNames(cols...)
	builder.SetValues(vals...)
	builder.SetWhere(gosql.Column(id).Equal(idValue))
	query, values := builder.Build()
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
	ctx context.Context, db *gosql.DB, idValue int64,
	id, table string,
) error {
	builder := db.Delete(table)
	builder.SetWhere(gosql.Column(id).Equal(idValue))
	query, values := builder.Build()
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
