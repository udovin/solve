package db

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"

	"github.com/udovin/gosql"
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

var testDB *gosql.DB

var sqliteCreateTables = []string{
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

var sqliteDropTables = []string{
	`DROP TABLE "test_event"`,
	`DROP TABLE "test_object"`,
}

var sqliteConfig = config.DB{
	Options: config.SQLiteOptions{Path: ":memory:"},
}

func testSetup(tb testing.TB, cfg config.DB, creates []string) {
	var err error
	testDB, err = cfg.Create()
	if err != nil {
		os.Exit(1)
	}
	for _, query := range creates {
		if _, err := testDB.Exec(query); err != nil {
			tb.Fatal(err)
		}
	}
}

func testTeardown(tb testing.TB, removes []string) {
	for _, query := range removes {
		if _, err := testDB.Exec(query); err != nil {
			tb.Fatal(err)
		}
	}
	_ = testDB.Close()
}

func testEventStore(t *testing.T, cfg config.DB, creates, removes []string) {
	testSetup(t, cfg, creates)
	defer testTeardown(t, removes)
	store := NewEventStore[testEvent]("id", "test_event", testDB)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Commit() }()
	ctx := gosql.WithTx(context.Background(), tx)
	events := []testEvent{
		{C: 8}, {C: 16}, {C: 5}, {C: 3},
		{testExtraEvent: testExtraEvent{A: "qwerty"}, C: 10},
	}
	for i, event := range events {
		created := event
		if err := store.CreateEvent(ctx, &created); err != nil {
			t.Fatal("Error:", err)
		}
		events[i].ID = created.EventID()
		if events[i].ID != int64(i+1) {
			t.Fatal()
		}
		if events[i] != created {
			t.Fatal()
		}
		id, err := store.LastEventID(ctx)
		if err != nil {
			t.Fatal("Error:", err)
		}
		if id != created.EventID() {
			t.Fatal()
		}
	}
	rows, err := store.LoadEvents(ctx, []EventRange{{Begin: 1, End: 6}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var createdEvents []testEvent
	for rows.Next() {
		createdEvents = append(createdEvents, rows.Event())
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(createdEvents, events) {
		t.Fatal()
	}
}

func TestSQLiteEventStore(t *testing.T) {
	testEventStore(t, sqliteConfig, sqliteCreateTables, sqliteDropTables)
}

func TestEventStoreClosed(t *testing.T) {
	testSetup(t, sqliteConfig, sqliteCreateTables)
	defer testTeardown(t, sqliteDropTables)
	store := NewEventStore[testEvent]("id", "test_event", testDB)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	ctx := gosql.WithTx(context.Background(), tx)
	if _, err := store.LastEventID(ctx); err != sql.ErrNoRows {
		t.Fatalf("Expected %v, got %v", sql.ErrNoRows, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := store.LastEventID(ctx); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
	if _, err := store.LoadEvents(ctx, []EventRange{{Begin: 1, End: 100}}); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
	event := testEvent{}
	if err := store.CreateEvent(ctx, &event); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
}
