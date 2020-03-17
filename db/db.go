// db provides implementation of generic object and event stores.
package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// Dialect represents SQL dialect.
type Dialect int

const (
	// SQLite represents SQLite.
	SQLite Dialect = 1
	// Postgres represents Postgres.
	Postgres Dialect = 2
)

func (d Dialect) String() string {
	switch d {
	case SQLite:
		return "SQLite"
	case Postgres:
		return "Postgres"
	default:
		return fmt.Sprintf("Dialect(%d)", d)
	}
}

func cloneRow(row interface{}) reflect.Value {
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

func scanRow(typ reflect.Type, rows *sql.Rows) (interface{}, error) {
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

func checkColumns(typ reflect.Type, rows *sql.Rows) error {
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
	recursive(typ)
	rowCols, err := rows.Columns()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(cols, rowCols) {
		return fmt.Errorf("result has invalid column sequence")
	}
	return nil
}

func prepareSelect(typ reflect.Type) string {
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

func prepareInsert(
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

func insertRow(
	tx *sql.Tx, row interface{}, id, table string, dialect Dialect,
) (interface{}, error) {
	clone := cloneRow(row)
	cols, keys, vals, idPtr := prepareInsert(clone, id)
	switch dialect {
	case Postgres:
		rows := tx.QueryRow(
			fmt.Sprintf(
				"INSERT INTO %q (%s) VALUES (%s) RETURNING %q",
				table, cols, keys, id,
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
				table, cols, keys,
			),
			vals...,
		)
		if err != nil {
			return nil, err
		}
		count, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if count != 1 {
			return nil, fmt.Errorf(
				"invalid amount of affected rows: %d", count,
			)
		}
		if *idPtr, err = res.LastInsertId(); err != nil {
			return nil, err
		}
	}
	return clone.Interface(), nil
}

func prepareUpdate(value reflect.Value, id string) (string, []interface{}) {
	var sets strings.Builder
	var vals []interface{}
	var idValue interface{}
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

func updateRow(
	tx *sql.Tx, row interface{}, id, table string,
) (interface{}, error) {
	clone := cloneRow(row)
	sets, vals := prepareUpdate(clone, id)
	res, err := tx.Exec(
		fmt.Sprintf(
			"UPDATE %q SET %s WHERE %q = $%d",
			table, sets, id, len(vals),
		),
		vals...,
	)
	if err != nil {
		return nil, err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, sql.ErrNoRows
	}
	return clone.Interface(), nil
}

func deleteRow(
	tx *sql.Tx, idValue int64, id, table string,
) error {
	res, err := tx.Exec(
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
