package db

import (
	"database/sql"
	"os"
	"reflect"
	"testing"

	"github.com/udovin/solve/config"
)

type testExtraEvent struct {
	A string `db:"a"`
	B int    `db:"b"`
}

type testEvent struct {
	mockEvent
	testExtraEvent
	C int `db:"c"`
}

var testDB *sql.DB

var createTables = []string{
	`CREATE TABLE "test_object"
(
	"id" INTEGER PRIMARY KEY,
	"a" VARCHAR(255),
	"b" INTEGER,
	"c" INTEGER
)`,
	`CREATE TABLE "test_event"
(
	"id" INTEGER PRIMARY KEY,
	"time" BIGINT,
	"a" VARCHAR(255),
	"b" INTEGER,
	"c" INTEGER,
	"d" INTEGER
)`,
}

var dropTables = []string{
	`DROP TABLE "test_event"`,
	`DROP TABLE "test_object"`,
}

func testSetup(tb testing.TB) {
	cfg := config.DB{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: "?mode=memory"},
	}
	var err error
	testDB, err = cfg.Create()
	if err != nil {
		os.Exit(1)
	}
	for _, query := range createTables {
		if _, err := testDB.Exec(query); err != nil {
			tb.Fatal(err)
		}
	}
}

func testTeardown(tb testing.TB) {
	for _, query := range dropTables {
		if _, err := testDB.Exec(query); err != nil {
			tb.Fatal(err)
		}
	}
	_ = testDB.Close()
}

func TestEventStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := NewEventStore(testEvent{}, "id", "test_event", SQLite)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Commit() }()
	events := []testEvent{
		{C: 8}, {C: 16}, {C: 5}, {C: 3},
		{testExtraEvent: testExtraEvent{A: "qwerty"}, C: 10},
	}
	for i, event := range events {
		created, err := store.CreateEvent(tx, event)
		if err != nil {
			t.Fatal(err)
		}
		events[i].ID = created.EventID()
		if events[i].ID != int64(i+1) {
			t.Fatal()
		}
		if events[i] != created.(testEvent) {
			t.Fatal()
		}
	}
	rows, err := store.LoadEvents(tx, 1, 6)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var createdEvents []testEvent
	for rows.Next() {
		createdEvents = append(createdEvents, rows.Event().(testEvent))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(createdEvents, events) {
		t.Fatal()
	}
}

func TestEventStoreClosed(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := NewEventStore(testEvent{}, "id", "test_event", SQLite)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := store.LoadEvents(tx, 1, 100); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
	if _, err := store.CreateEvent(tx, testEvent{}); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
}
