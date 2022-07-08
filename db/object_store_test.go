package db

import (
	"context"
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
	ID int64 `db:"id"`
	C  int   `db:"c"`
}

func (o testObject) ObjectID() int64 {
	return o.ID
}

func (o *testObject) SetObjectID(id int64) {
	o.ID = id
}

func testSetupObjectStore(t testing.TB, store ObjectStore[testObject, *testObject]) []testObject {
	objects := []testObject{
		{C: 8}, {C: 16}, {C: 5}, {C: 3},
		{testExtraObject: testExtraObject{A: "qwerty"}, C: 10},
	}
	for i, object := range objects {
		err := store.CreateObject(context.Background(), &object)
		if err != nil {
			t.Fatal(err)
		}
		objects[i].ID = object.ObjectID()
		if objects[i].ID != int64(i+1) {
			t.Fatal()
		}
		if objects[i] != object {
			t.Fatal()
		}
	}
	return objects
}

func TestObjectStore(t *testing.T) {
	testSetup(t, sqliteConfig, sqliteCreateTables)
	defer testTeardown(t, sqliteDropTables)
	store := NewObjectStore[testObject]("id", "test_object", testDB)
	objects := testSetupObjectStore(t, store)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Commit() }()
	ctx := WithTx(context.Background(), tx)
	rows, err := store.LoadObjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var createdObjects []testObject
	for rows.Next() {
		createdObjects = append(createdObjects, rows.Row())
	}
	if err := rows.Err(); err != nil {
		t.Fatal("Error:", err)
	}
	if !reflect.DeepEqual(createdObjects, objects) {
		t.Fatalf("Expected %v, got %v", objects, createdObjects)
	}
	objects[0].A = "Updated text"
	updatedObject := objects[0]
	if err := store.UpdateObject(ctx, &updatedObject); err != nil {
		t.Fatal("Error:", err)
	}
	if updatedObject != objects[0] {
		t.Fatalf("Expected %v, got %v", objects[0], updatedObject)
	}
	unknownObject := testObject{ID: 10000}
	if err := store.UpdateObject(ctx, &unknownObject); err != sql.ErrNoRows {
		t.Fatalf("Expected %v, got %v", sql.ErrNoRows, err)
	}
	if err := store.DeleteObject(ctx, 10000); err != sql.ErrNoRows {
		t.Fatalf("Expected %v, got %v", sql.ErrNoRows, err)
	}
	if err := store.DeleteObject(ctx, objects[0].ID); err != nil {
		t.Fatal("Error:", err)
	}
	if err := store.DeleteObject(ctx, objects[0].ID); err != sql.ErrNoRows {
		t.Fatalf("Expected %v, got %v", sql.ErrNoRows, err)
	}
}

func TestObjectStoreClosed(t *testing.T) {
	testSetup(t, sqliteConfig, sqliteCreateTables)
	defer testTeardown(t, sqliteDropTables)
	store := NewObjectStore[testObject]("id", "test_object", testDB)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithTx(context.Background(), tx)
	if err := tx.Rollback(); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := store.LoadObjects(ctx); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
	if _, err := store.FindObjects(ctx, nil); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
	var object testObject
	if err := store.CreateObject(ctx, &object); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
	if err := store.UpdateObject(ctx, &object); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
	if err := store.DeleteObject(ctx, 1); err != sql.ErrTxDone {
		t.Fatalf("Expected %v, got %v", sql.ErrTxDone, err)
	}
}

func TestObjectStoreLoadObjectsFail(t *testing.T) {
	testSetup(t, sqliteConfig, sqliteCreateTables)
	defer testTeardown(t, sqliteDropTables)
	store := NewObjectStore[testObject]("id", "test_object", testDB)
	objects := testSetupObjectStore(t, store)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Commit() }()
	ctx := WithTx(context.Background(), tx)
	rows, err := store.LoadObjects(ctx)
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

func TestObjectStoreFindObjectsFail(t *testing.T) {
	testSetup(t, sqliteConfig, sqliteCreateTables)
	defer testTeardown(t, sqliteDropTables)
	store := NewObjectStore[testObject]("id", "test_object", testDB)
	objects := testSetupObjectStore(t, store)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Commit() }()
	ctx := WithTx(context.Background(), tx)
	rows, err := store.FindObjects(ctx, nil)
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
