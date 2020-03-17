package db

import (
	"testing"
)

func TestDialectString(t *testing.T) {
	if v := SQLite.String(); v != "SQLite" {
		t.Fatalf("Invalid dialect string: %q", v)
	}
	if v := Postgres.String(); v != "Postgres" {
		t.Fatalf("Invalid dialect string: %q", v)
	}
	if v := Dialect(1000).String(); v != "Dialect(1000)" {
		t.Fatalf("Invalid dialect string: %q", v)
	}
}
