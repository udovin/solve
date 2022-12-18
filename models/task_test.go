package models

import (
	"database/sql"
	"fmt"
	"testing"
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
			`"expire_time" integer)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "task_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_kind" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"event_account_id" integer NULL,` +
			`"id" integer NOT NULL,` +
			`"status" integer NOT NULL,` +
			`"kind" integer NOT NULL,` +
			`"config" blob NOT NULL,` +
			`"state" blob NOT NULL,` +
			`"expire_time" integer)`,
	)
	return err
}

func (t *taskStoreTest) newStore() Store {
	return NewTaskStore(testDB, "task", "task_event")
}

func (t *taskStoreTest) newObject() Object {
	return Task{}
}

func (t *taskStoreTest) createObject(
	s Store, tx *sql.Tx, o Object,
) (Object, error) {
	task := o.(Task)
	err := s.(*TaskStore).Create(wrapContext(tx), &task)
	return task, err
}

func (t *taskStoreTest) updateObject(
	s Store, tx *sql.Tx, o Object,
) (Object, error) {
	return o, s.(*TaskStore).Update(wrapContext(tx), o.(Task))
}

func (t *taskStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*TaskStore).Delete(wrapContext(tx), id)
}

func TestTaskStatus(t *testing.T) {
	//lint:ignore S1025 Used for tests.
	if s := fmt.Sprintf("%s", QueuedTask); s != "queued" {
		t.Errorf("Expected %q, got %q", "queued", s)
	}
	//lint:ignore S1025 Used for tests.
	if s := fmt.Sprintf("%s", RunningTask); s != "running" {
		t.Errorf("Expected %q, got %q", "running", s)
	}
	//lint:ignore S1025 Used for tests.
	if s := fmt.Sprintf("%s", SucceededTask); s != "succeeded" {
		t.Errorf("Expected %q, got %q", "succeeded", s)
	}
	//lint:ignore S1025 Used for tests.
	if s := fmt.Sprintf("%s", FailedTask); s != "failed" {
		t.Errorf("Expected %q, got %q", "failed", s)
	}
	//lint:ignore S1025 Used for tests.
	if s := fmt.Sprintf("%s", TaskStatus(-1)); s != "TaskStatus(-1)" {
		t.Errorf("Expected %q, got %q", "TaskStatus(-1)", s)
	}
	text, err := SucceededTask.MarshalText()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if string(text) != "succeeded" {
		t.Errorf("Expected %q, got %q", "succeeded", string(text))
	}
}

func TestTaskKind(t *testing.T) {
	//lint:ignore S1025 Used for tests.
	if s := fmt.Sprintf("%s", JudgeSolutionTask); s != "judge_solution" {
		t.Errorf("Expected %q, got %q", "judge_solution", s)
	}
	//lint:ignore S1025 Used for tests.
	if s := fmt.Sprintf("%s", TaskKind(-1)); s != "TaskKind(-1)" {
		t.Errorf("Expected %q, got %q", "TaskKind(-1)", s)
	}
	text, err := JudgeSolutionTask.MarshalText()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if string(text) != "judge_solution" {
		t.Errorf("Expected %q, got %q", "judge_solution", string(text))
	}
}

func TestTaskStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&taskStoreTest{}}
	tester.Test(t)
}
