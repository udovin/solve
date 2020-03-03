package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type roleEdgeManagerTest struct{}

func (t *roleEdgeManagerTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "role_edge" (` +
			`"id" integer PRIMARY KEY,` +
			`"role_id" integer NOT NULL,` +
			`"child_id" integer NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "role_edge_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"role_id" integer NOT NULL,` +
			`"child_id" integer NOT NULL)`,
	)
	return err
}

func (t *roleEdgeManagerTest) newManager() Manager {
	return NewRoleEdgeManager("role_edge", "role_edge_event", db.SQLite)
}

func (t *roleEdgeManagerTest) newObject() db.Object {
	return RoleEdge{}
}

func (t *roleEdgeManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*RoleEdgeManager).CreateTx(tx, o.(RoleEdge))
}

func (t *roleEdgeManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*RoleEdgeManager).UpdateTx(tx, o.(RoleEdge))
}

func (t *roleEdgeManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*RoleEdgeManager).DeleteTx(tx, id)
}

func TestRoleEdgeManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&roleEdgeManagerTest{}}
	tester.Test(t)
}
