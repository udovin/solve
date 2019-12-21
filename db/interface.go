package db

import (
	"database/sql"
	"time"
)

// Object represents object from store
type Object interface {
	// ObjectID should return id of object
	ObjectID() int64
}

// ObjectReader represents reader for objects
type ObjectReader interface {
	// Next should read next object and return true if object exists
	Next() bool
	// Object should return current object
	Object() Object
	// Close should close reader
	Close() error
	// Err should return error that occurred during reading
	Err() error
}

// ObjectROStore represents persistent store for objects
type ObjectROStore interface {
	LoadObjects(tx *sql.Tx) (ObjectReader, error)
	FindObject(tx *sql.Tx, id int64) (Object, error)
}

type ObjectStore interface {
	ObjectROStore
	CreateObject(tx *sql.Tx, object Object) (Object, error)
	UpdateObject(tx *sql.Tx, object Object) (Object, error)
	DeleteObject(tx *sql.Tx, id int64) (Object, error)
}

// Event represents a change
type Event interface {
	EventID() int64
	EventTime() time.Time
}

// EventReader represents reader for events
type EventReader interface {
	// Next should read next change and return true if change exists
	Next() bool
	// Event should return current change
	Event() Event
	// Close should close reader
	Close() error
	// Err should return error that occurred during reading
	Err() error
}

// EventROStore represents persistent store for event
type EventROStore interface {
	LoadEvents(tx *sql.Tx, begin, end int64) (EventReader, error)
}

type EventStore interface {
	EventROStore
	CreateEvent(tx *sql.Tx, event Event) (Event, error)
}
