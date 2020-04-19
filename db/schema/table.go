package schema

import (
	"fmt"
	"strings"

	"github.com/udovin/solve/db"
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

// BuildSQL returns SQL in specified dialect.
func (c Column) BuildSQL(d db.Dialect) (string, error) {
	switch c.Type {
	case Int64:
		typeName := "bigint"
		if c.PrimaryKey {
			if d == db.SQLite {
				// SQLite does not support bigint primary keys.
				typeName = "integer"
			}
			if d == db.Postgres && c.AutoIncrement {
				// Postgres has special type for autoincrement columns.
				typeName = "bigserial"
			}
			typeName += " PRIMARY KEY"
			if c.AutoIncrement && d == db.SQLite {
				// AutoIncrement columns for SQLite should be marked
				// as autoincrement using following keyword.
				typeName += " AUTOINCREMENT"
			}
		} else if !c.Nullable {
			typeName += " NOT NULL"
		}
		return fmt.Sprintf("%q %s", c.Name, typeName), nil
	case String:
		typeName := "text"
		if !c.Nullable {
			typeName += " NOT NULL"
		}
		return fmt.Sprintf("%q %s", c.Name, typeName), nil
	default:
		return "", fmt.Errorf("unsupported column type: %v", c.Type)
	}
}

// Table represents table.
type Table struct {
	Name    string
	Columns []Column
}

// BuildCreateSQL returns create SQL query in specified dialect.
func (t Table) BuildCreateSQL(d db.Dialect) (string, error) {
	var query strings.Builder
	query.WriteString(fmt.Sprintf("CREATE TABLE %q(", t.Name))
	for i, column := range t.Columns {
		if i > 0 {
			query.WriteRune(',')
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
