package db

import (
	"database/sql"
	"time"
)

type Object interface {
	ObjectID() int64
}

type ObjectReader interface {
	Next() bool
	Object() Object
	Close() error
	Err() error
}

type ObjectStore interface {
	LoadObjects(tx *sql.Tx) (ObjectReader, error)
	FindObject(tx *sql.Tx, id int64) (Object, error)
	CreateObject(tx *sql.Tx, object Object) (Object, error)
	UpdateObject(tx *sql.Tx, object Object) (Object, error)
	DeleteObject(tx *sql.Tx, id int64) (Object, error)
}

type Change interface {
	ChangeID() int64
	ChangeTime() time.Time
}

type ChangeReader interface {
	Next() bool
	Change() Change
	Close() error
	Err() error
}

type ChangeStore interface {
	LoadChanges(tx *sql.Tx, begin, end int64) (ChangeReader, error)
	CreateChange(tx *sql.Tx, change Change) (Change, error)
}
