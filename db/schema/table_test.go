package schema

import (
	"testing"

	"github.com/udovin/gosql"
)

func TestColumnInt64(t *testing.T) {
	// PrimaryKey AutoIncrement Int64 column.
	c1 := Column{
		Name:          "test1",
		Type:          Int64,
		PrimaryKey:    true,
		AutoIncrement: true,
	}
	check := func(c Column, dialect gosql.Dialect, expected string) {
		if sql, err := c.BuildSQL(dialect); err != nil {
			t.Fatal("Error:", err)
		} else if sql != expected {
			t.Fatal("Wrong SQL:", sql)
		}
	}
	// Note that SQLite does not support bigint as primary key.
	check(c1, gosql.SQLiteDialect, `"test1" integer PRIMARY KEY AUTOINCREMENT`)
	check(c1, gosql.PostgresDialect, `"test1" bigserial PRIMARY KEY`)
	// PrimaryKey Int64 column.
	c2 := Column{Name: "test2", Type: Int64, PrimaryKey: true}
	check(c2, gosql.SQLiteDialect, `"test2" integer PRIMARY KEY`)
	check(c2, gosql.PostgresDialect, `"test2" bigint PRIMARY KEY`)
	// Int64 column.
	c3 := Column{Name: "test3", Type: Int64}
	check(c3, gosql.SQLiteDialect, `"test3" bigint NOT NULL`)
	check(c3, gosql.PostgresDialect, `"test3" bigint NOT NULL`)
	// Int64 column.
	c4 := Column{Name: "test4", Type: Int64, Nullable: true}
	check(c4, gosql.SQLiteDialect, `"test4" bigint`)
	check(c4, gosql.PostgresDialect, `"test4" bigint`)
}

func TestColumnString(t *testing.T) {
	c1 := Column{Name: "test1", Type: String}
	// Check for SQLite.
	if sql, err := c1.BuildSQL(gosql.SQLiteDialect); err != nil {
		t.Fatal("Error:", err)
	} else if sql != `"test1" text NOT NULL` {
		t.Fatal("Wrong SQL:", sql)
	}
	// Check for Postgres.
	if sql, err := c1.BuildSQL(gosql.PostgresDialect); err != nil {
		t.Fatal("Error:", err)
	} else if sql != `"test1" text NOT NULL` {
		t.Fatal("Wrong SQL:", sql)
	}
	// Nullable column.
	c2 := Column{Name: "test2", Type: String, Nullable: true}
	// Check for SQLite.
	if sql, err := c2.BuildSQL(gosql.SQLiteDialect); err != nil {
		t.Fatal("Error:", err)
	} else if sql != `"test2" text` {
		t.Fatal("Wrong SQL:", sql)
	}
	// Check for Postgres.
	if sql, err := c2.BuildSQL(gosql.PostgresDialect); err != nil {
		t.Fatal("Error:", err)
	} else if sql != `"test2" text` {
		t.Fatal("Wrong SQL:", sql)
	}
}

func TestColumnJSON(t *testing.T) {
	c1 := Column{Name: "test1", Type: JSON}
	// Check for SQLite.
	if sql, err := c1.BuildSQL(gosql.SQLiteDialect); err != nil {
		t.Fatal("Error:", err)
	} else if sql != `"test1" blob NOT NULL` {
		t.Fatal("Wrong SQL:", sql)
	}
	// Check for Postgres.
	if sql, err := c1.BuildSQL(gosql.PostgresDialect); err != nil {
		t.Fatal("Error:", err)
	} else if sql != `"test1" jsonb NOT NULL` {
		t.Fatal("Wrong SQL:", sql)
	}
	// Nullable column.
	c2 := Column{Name: "test2", Type: JSON, Nullable: true}
	// Check for SQLite.
	if sql, err := c2.BuildSQL(gosql.SQLiteDialect); err != nil {
		t.Fatal("Error:", err)
	} else if sql != `"test2" blob` {
		t.Fatal("Wrong SQL:", sql)
	}
	// Check for Postgres.
	if sql, err := c2.BuildSQL(gosql.PostgresDialect); err != nil {
		t.Fatal("Error:", err)
	} else if sql != `"test2" jsonb` {
		t.Fatal("Wrong SQL:", sql)
	}
}

func TestColumnInvalid(t *testing.T) {
	c1 := Column{Name: "test", Type: 228}
	if _, err := c1.BuildSQL(gosql.SQLiteDialect); err == nil {
		t.Fatal("Expected error")
	}
}

func TestTableSimple(t *testing.T) {
	t1 := Table{
		Name: "test_table",
		Columns: []Column{
			{Name: "id", Type: Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: String},
		},
	}
	t1SQLite := `CREATE TABLE "test_table"("id" integer PRIMARY KEY AUTOINCREMENT,"name" text NOT NULL)`
	if sql, err := t1.BuildCreateSQL(gosql.SQLiteDialect); err != nil {
		t.Fatal("Error:", err)
	} else if sql != t1SQLite {
		t.Fatal("Wrong SQL:", sql)
	}
	t1Postgres := `CREATE TABLE "test_table"("id" bigserial PRIMARY KEY,"name" text NOT NULL)`
	if sql, err := t1.BuildCreateSQL(gosql.PostgresDialect); err != nil {
		t.Fatal("Error:", err)
	} else if sql != t1Postgres {
		t.Fatal("Wrong SQL:", sql)
	}
}

func TestTableInvalidColumn(t *testing.T) {
	t1 := Table{
		Name: "test_table",
		Columns: []Column{
			{Name: "id", Type: 228},
		},
	}
	if _, err := t1.BuildCreateSQL(gosql.SQLiteDialect); err == nil {
		t.Fatal("Expected error")
	}
	if _, err := t1.BuildCreateSQL(gosql.PostgresDialect); err == nil {
		t.Fatal("Expected error")
	}
}
