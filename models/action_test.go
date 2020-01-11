package models

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/udovin/solve/db"
)

type actionManagerTest struct{}

func (t *actionManagerTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "action" (` +
			`"id" integer PRIMARY KEY,` +
			`"status" integer NOT NULL,` +
			`"type" integer NOT NULL,` +
			`"config" blob NOT NULL,` +
			`"state" blob NOT NULL,` +
			`"expire_time" integer NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "action_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"status" integer NOT NULL,` +
			`"type" integer NOT NULL,` +
			`"config" blob NOT NULL,` +
			`"state" blob NOT NULL,` +
			`"expire_time" integer NOT NULL)`,
	)
	return err
}

func (t *actionManagerTest) newManager() Manager {
	return NewActionManager("action", "action_event", db.SQLite)
}

func (t *actionManagerTest) newObject() db.Object {
	return Action{}
}

func (t *actionManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*ActionManager).CreateTx(tx, o.(Action))
}

func (t *actionManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*ActionManager).UpdateTx(tx, o.(Action))
}

func (t *actionManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*ActionManager).DeleteTx(tx, id)
}

func TestActionStatus(t *testing.T) {
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
	if s := fmt.Sprintf("%s", ActionStatus(-1)); s != "ActionStatus(-1)" {
		t.Errorf("Expected %q, got %q", "ActionStatus(-1)", s)
	}
	text, err := Succeeded.MarshalText()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if string(text) != "Succeeded" {
		t.Errorf("Expected %q, got %q", "Succeeded", string(text))
	}
}

func TestActionType(t *testing.T) {
	if s := fmt.Sprintf("%s", JudgeSolution); s != "JudgeSolution" {
		t.Errorf("Expected %q, got %q", "JudgeSolution", s)
	}
	if s := fmt.Sprintf("%s", ActionType(-1)); s != "ActionType(-1)" {
		t.Errorf("Expected %q, got %q", "ActionType(-1)", s)
	}
	text, err := JudgeSolution.MarshalText()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if string(text) != "JudgeSolution" {
		t.Errorf("Expected %q, got %q", "JudgeSolution", string(text))
	}
}

func TestActionManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&actionManagerTest{}}
	tester.Test(t)
}
