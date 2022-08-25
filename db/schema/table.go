package schema

import (
	"fmt"
	"strings"

	"github.com/udovin/gosql"
)

// Type represents type of column.
type Type int

const (
	// Int64 represents golang int64 type in SQL.
	Int64 Type = 1 + iota
	// String represents golang string type in SQL.
	String
	// JSON represents models.JSON type in SQL.
	JSON
)

// Column represents table column with parameters.
type Column struct {
	Name          string
	Type          Type
	PrimaryKey    bool
	AutoIncrement bool
	Nullable      bool
}

const (
	// Common strings for SQL.
	suffixPrimaryKey = " PRIMARY KEY"
	suffixNotNULL    = " NOT NULL"
)

// int64BuildSQL returns SQL string for Int64 column.
func (c Column) int64BuildSQL(d gosql.Dialect) (string, error) {
	typeName := "bigint"
	if c.PrimaryKey {
		if d == gosql.SQLiteDialect {
			// SQLite does not support bigint primary keys.
			typeName = "integer"
		}
		if d == gosql.PostgresDialect && c.AutoIncrement {
			// Postgres has special type for autoincrement columns.
			typeName = "bigserial"
		}
		typeName += suffixPrimaryKey
		if c.AutoIncrement && d == gosql.SQLiteDialect {
			// AutoIncrement columns for SQLite should be marked
			// as autoincrement using following keyword.
			typeName += " AUTOINCREMENT"
		}
	} else if !c.Nullable {
		typeName += suffixNotNULL
	}
	return fmt.Sprintf("%q %s", c.Name, typeName), nil
}

// BuildSQL returns SQL in specified dialect.
func (c Column) BuildSQL(d gosql.Dialect) (string, error) {
	switch c.Type {
	case Int64:
		return c.int64BuildSQL(d)
	case String:
		typeName := "text"
		if !c.Nullable {
			typeName += suffixNotNULL
		}
		return fmt.Sprintf("%q %s", c.Name, typeName), nil
	case JSON:
		typeName := "blob"
		if d == gosql.PostgresDialect {
			// Postgres has special types for JSON: json and jsonb.
			// We prefer jsonb over json because it is more efficient.
			typeName = "jsonb"
		}
		if !c.Nullable {
			typeName += suffixNotNULL
		}
		return fmt.Sprintf("%q %s", c.Name, typeName), nil
	default:
		return "", fmt.Errorf("unsupported column type: %v", c.Type)
	}
}

type Operation interface {
	BuildApply(gosql.Dialect) (string, error)
	BuildUnapply(gosql.Dialect) (string, error)
}

// CreateTable represents create table query.
type CreateTable struct {
	Name    string
	Columns []Column
	Strict  bool
}

// BuildCreateSQL returns create SQL query in specified dialect.
func (q CreateTable) BuildApply(d gosql.Dialect) (string, error) {
	var query strings.Builder
	query.WriteString("CREATE TABLE ")
	if !q.Strict {
		query.WriteString("IF NOT EXISTS ")
	}
	query.WriteString(fmt.Sprintf("%q (", q.Name))
	for i, column := range q.Columns {
		if i > 0 {
			query.WriteString(", ")
		}
		sql, err := column.BuildSQL(d)
		if err != nil {
			return "", err
		}
		query.WriteString(sql)
	}
	query.WriteRune(')')
	return query.String(), nil
}

func (q CreateTable) BuildUnapply(d gosql.Dialect) (string, error) {
	var query strings.Builder
	query.WriteString("DROP TABLE ")
	if !q.Strict {
		query.WriteString("IF EXISTS ")
	}
	query.WriteString(fmt.Sprintf("%q", q.Name))
	return query.String(), nil
}
