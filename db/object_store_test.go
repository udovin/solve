package db

import (
	"database/sql"
	"reflect"
	"testing"
)

type testExtraObject struct {
	A string `db:"a"`
	B int    `db:"b"`
}

type testObject struct {
	testExtraObject
	Id int64 `db:"id"`
	C  int   `db:"c"`
}

func (o testObject) ObjectId() int64 {
	return o.Id
}

func testSetupObjectStore(t testing.TB, store ObjectStore) []testObject {
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Commit() }()
	objects := []testObject{
		{C: 8}, {C: 16}, {C: 5}, {C: 3},
		{testExtraObject: testExtraObject{A: "qwerty"}, C: 10},
	}
	for i, object := range objects {
		created, err := store.CreateObject(tx, object)
		if err != nil {
			t.Fatal(err)
		}
		objects[i].Id = created.ObjectId()
		if objects[i].Id != int64(i+1) {
			t.Fatal()
		}
		if objects[i] != created.(testObject) {
			t.Fatal()
		}
	}
	return objects
}

func TestObjectStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := NewObjectStore(testObject{}, "id", "test_object", SQLite)
	objects := testSetupObjectStore(t, store)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Commit() }()
	rows, err := store.LoadObjects(tx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var createdObjects []testObject
	for rows.Next() {
		createdObjects = append(createdObjects, rows.Object().(testObject))
	}
	if err := rows.Err(); err != nil {
		t.Fatal("Error:", err)
	}
	if !reflect.DeepEqual(createdObjects, objects) {
		t.Fatalf("Expected %v, got %v", objects, createdObjects)
	}
	objects[0].A = "Updated text"
	updatedObject, err := store.UpdateObject(tx, objects[0])
	if err != nil {
		t.Fatal("Error:", err)
	}
	if updatedObject != objects[0] {
		t.Fatalf("Expected %v, got %v", objects[0], updatedObject)
	}
	if _, err := store.UpdateObject(
		tx, testObject{Id: 10000},
	); err != sql.ErrNoRows {
		t.Fatalf("Expected %v, got %v", sql.ErrNoRows, err)
	}
	if err := store.DeleteObject(tx, 10000); err != sql.ErrNoRows {
		t.Fatalf("Expected %v, got %v", sql.ErrNoRows, err)
	}
	if err := store.DeleteObject(tx, objects[0].Id); err != nil {
		t.Fatal("Error:", err)
	}
	if err := store.DeleteObject(tx, objects[0].Id); err != sql.ErrNoRows {
		t.Fatalf("Expected %v, got %v", sql.ErrNoRows, err)
	}
}

func TestObjectStoreClosed(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := NewObjectStore(testObject{}, "id", "test_object", SQLite)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := store.LoadObjects(tx); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
	if _, err := store.CreateObject(tx, testObject{}); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
	if _, err := store.UpdateObject(tx, testObject{}); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
	if err := store.DeleteObject(tx, 1); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
}

func TestObjectStoreLoadObjectsFail(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := NewObjectStore(testObject{}, "id", "test_object", SQLite)
	objects := testSetupObjectStore(t, store)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Commit() }()
	rows, err := store.LoadObjects(tx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for i := 1; i < len(objects); i++ {
		if !rows.Next() {
			t.Fatal("Expected next object")
		}
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal("Error:", err)
	}
	if rows.Next() {
		t.Fatal("Expected end of rows")
	}
}
