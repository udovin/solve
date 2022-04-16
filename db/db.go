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
}

func (r *rowReader[T]) Next() bool {
	if !r.rows.Next() {
		return false
	}
	r.err = scanRow(&r.row, r.rows)
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

func cloneRow(row any) reflect.Value {
	clone := reflect.New(reflect.TypeOf(row)).Elem()
	var recursive func(row, clone reflect.Value)
	recursive = func(row, clone reflect.Value) {
		t := row.Type()
		for i := 0; i < t.NumField(); i++ {
			if _, ok := t.Field(i).Tag.Lookup("db"); ok {
				clone.Field(i).Set(row.Field(i))
			} else if t.Field(i).Anonymous {
				recursive(row.Field(i), clone.Field(i))
			}
		}
	}
	recursive(reflect.ValueOf(row), clone)
	return clone
}

func scanRow[T any](row *T, rows *sql.Rows) error {
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
	return rows.Scan(fields...)
}

func checkColumns(rows *sql.Rows, cols []string) error {
	rowCols, err := rows.Columns()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(cols, rowCols) {
		return fmt.Errorf("result has invalid column sequence")
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

func prepareSelect[T any]() string {
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
	var object T
	recursive(reflect.TypeOf(object))
	return cols.String()
}

func prepareInsert(
	value reflect.Value, id string,
) (string, string, []any, *int64) {
	var cols strings.Builder
	var keys strings.Builder
	var vals []any
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

func insertRow[T any](
	ctx context.Context, db *gosql.DB, row *T,
	id, table string,
) error {
	cols, keys, vals, idPtr := prepareInsert(reflect.ValueOf(row).Elem(), id)
	tx := GetRunner(ctx, db)
	switch db.Dialect() {
	case gosql.PostgresDialect:
		rows := tx.QueryRowContext(
			ctx,
			fmt.Sprintf(
				"INSERT INTO %q (%s) VALUES (%s) RETURNING %q",
				table, cols, keys, id,
			),
			vals...,
		)
		if err := rows.Scan(idPtr); err != nil {
			return err
		}
	default:
		res, err := tx.ExecContext(
			ctx,
			fmt.Sprintf(
				"INSERT INTO %q (%s) VALUES (%s)",
				table, cols, keys,
			),
			vals...,
		)
		if err != nil {
			return err
		}
		count, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf(
				"invalid amount of affected rows: %d", count,
			)
		}
		if *idPtr, err = res.LastInsertId(); err != nil {
			return err
		}
	}
	return nil
}

func prepareUpdate(value reflect.Value, id string) (string, []any) {
	var sets strings.Builder
	var vals []any
	var idValue any
	var it int
	var recursive func(reflect.Value)
	recursive = func(v reflect.Value) {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			if db, ok := t.Field(i).Tag.Lookup("db"); ok {
				name := strings.Split(db, ",")[0]
				if name == id {
					idValue = v.Field(i).Interface()
					continue
				}
				if it > 0 {
					sets.WriteRune(',')
				}
				it++
				sets.WriteString(fmt.Sprintf("%q = $%d", name, it))
				vals = append(vals, v.Field(i).Interface())
			} else if t.Field(i).Anonymous {
				recursive(v.Field(i))
			}
		}
	}
	recursive(value)
	vals = append(vals, idValue)
	return sets.String(), vals
}

func updateRow[T any](
	ctx context.Context, db *gosql.DB, row T,
	id, table string,
) error {
	sets, vals := prepareUpdate(reflect.ValueOf(row), id)
	tx := GetRunner(ctx, db)
	res, err := tx.ExecContext(
		ctx,
		fmt.Sprintf(
			"UPDATE %q SET %s WHERE %q = $%d",
			table, sets, id, len(vals),
		),
		vals...,
	)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func deleteRow(
	ctx context.Context, db *gosql.DB,
	idValue int64, id, table string,
) error {
	tx := GetRunner(ctx, db)
	res, err := tx.ExecContext(
		ctx,
		fmt.Sprintf(
			"DELETE FROM %q WHERE %q = $1",
			table, id,
		),
		idValue,
	)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return sql.ErrNoRows
	}
	return nil
}
