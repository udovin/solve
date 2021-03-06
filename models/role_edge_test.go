package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type roleEdgeStoreTest struct{}

func (t *roleEdgeStoreTest) prepareDB(tx *sql.Tx) error {
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

func (t *roleEdgeStoreTest) newStore() Store {
	return NewRoleEdgeStore("role_edge", "role_edge_event", db.SQLite)
}

func (t *roleEdgeStoreTest) newObject() db.Object {
	return RoleEdge{}
}

func (t *roleEdgeStoreTest) createObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return s.(*RoleEdgeStore).CreateTx(tx, o.(RoleEdge))
}

func (t *roleEdgeStoreTest) updateObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, s.(*RoleEdgeStore).UpdateTx(tx, o.(RoleEdge))
}

func (t *roleEdgeStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*RoleEdgeStore).DeleteTx(tx, id)
}

func TestRoleEdgeStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&roleEdgeStoreTest{}}
	tester.Test(t)
}
