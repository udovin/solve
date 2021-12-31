package models

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/udovin/solve/db"
)

type taskStoreTest struct{}

func (t *taskStoreTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "task" (` +
			`"id" integer PRIMARY KEY,` +
			`"status" integer NOT NULL,` +
			`"kind" integer NOT NULL,` +
			`"config" blob NOT NULL,` +
			`"state" blob NOT NULL,` +
			`"expire_time" integer NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "task_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"status" integer NOT NULL,` +
			`"kind" integer NOT NULL,` +
			`"config" blob NOT NULL,` +
			`"state" blob NOT NULL,` +
			`"expire_time" integer NOT NULL)`,
	)
	return err
}

func (t *taskStoreTest) newStore() Store {
	return NewTaskStore(testDB, "task", "task_event")
}

func (t *taskStoreTest) newObject() db.Object {
	return Task{}
}

func (t *taskStoreTest) createObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	task := o.(Task)
	err := s.(*TaskStore).CreateTx(tx, &task)
	return task, err
}

func (t *taskStoreTest) updateObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, s.(*TaskStore).UpdateTx(tx, o.(Task))
}

func (t *taskStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*TaskStore).DeleteTx(tx, id)
}

func TestTaskStatus(t *testing.T) {
	if s := fmt.Sprintf("%s", Queued); s != "Queued" {
		t.Errorf("Expected %q, got %q", "Queued", s)
	}
	if s := fmt.Sprintf("%s", Running); s != "Running" {
		t.Errorf("Expected %q, got %q", "Running", s)
	}
	if s := fmt.Sprintf("%s", Succeeded); s != "Succeeded" {
		t.Errorf("Expected %q, got %q", "Succeeded", s)
	}
	if s := fmt.Sprintf("%s", Failed); s != "Failed" {
		t.Errorf("Expected %q, got %q", "Failed", s)
	}
	if s := fmt.Sprintf("%s", TaskStatus(-1)); s != "TaskStatus(-1)" {
		t.Errorf("Expected %q, got %q", "TaskStatus(-1)", s)
	}
	text, err := Succeeded.MarshalText()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if string(text) != "Succeeded" {
		t.Errorf("Expected %q, got %q", "Succeeded", string(text))
	}
}

func TestTaskKind(t *testing.T) {
	if s := fmt.Sprintf("%s", JudgeSolution); s != "JudgeSolution" {
		t.Errorf("Expected %q, got %q", "JudgeSolution", s)
	}
	if s := fmt.Sprintf("%s", TaskKind(-1)); s != "TaskKind(-1)" {
		t.Errorf("Expected %q, got %q", "TaskKind(-1)", s)
	}
	text, err := JudgeSolution.MarshalText()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if string(text) != "JudgeSolution" {
		t.Errorf("Expected %q, got %q", "JudgeSolution", string(text))
	}
}

func TestTaskStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&taskStoreTest{}}
	tester.Test(t)
}
