package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/udovin/solve/db"
)

// EventType represents type of object event
type EventType int8

const (
	// CreateEvent means that this is event of object creation
	CreateEvent EventType = 1
	// DeleteEvent means that this is event of object deletion
	DeleteEvent EventType = 2
	// UpdateEvent means that this is event of object modification
	UpdateEvent EventType = 3
)

// ObjectEvent represents event for object
type ObjectEvent interface {
	db.Event
	// EventType should return type of object event
	EventType() EventType
	// Object should return struct with object data
	Object() db.Object
	// WithObject should return copy of event with replaced object
	WithObject(db.Object) ObjectEvent
}

// baseEvent represents base for all events
type baseEvent struct {
	// BaseEventId contains event id
	BaseEventId int64 `db:"event_id" json:"EventId"`
	// BaseEventType contains type of event
	BaseEventType EventType `db:"event_type" json:"EventType"`
	// BaseEventTime contains event type
	BaseEventTime int64 `db:"event_time" json:"EventTime"`
}

// EventId returns id of this event
func (e baseEvent) EventId() int64 {
	return e.BaseEventId
}

// EventTime returns time of this event
func (e baseEvent) EventTime() time.Time {
	return time.Unix(e.BaseEventTime, 0)
}

// EventType returns type of this event
func (e baseEvent) EventType() EventType {
	return e.BaseEventType
}

// makeBaseEvent creates baseEvent with specified type
func makeBaseEvent(t EventType) baseEvent {
	return baseEvent{BaseEventType: t, BaseEventTime: time.Now().Unix()}
}

type baseManagerImpl interface {
	reset()
	addObject(o db.Object)
	onCreateObject(o db.Object)
	onUpdateObject(o db.Object)
	onDeleteObject(o db.Object)
	updateSchema(tx *sql.Tx, version int) (int, error)
}

type Manager interface {
	Init(tx *sql.Tx) error
	Sync(tx *sql.Tx) error
}

type baseManager struct {
	objects  db.ObjectStore
	events   db.EventStore
	consumer db.EventConsumer
	impl     baseManagerImpl
	mutex    sync.RWMutex
}

func (m *baseManager) Init(tx *sql.Tx) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	rows, err := m.objects.LoadObjects(tx)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()
	m.impl.reset()
	m.consumer = db.NewEventConsumer(m.events, 1)
	for rows.Next() {
		m.impl.addObject(rows.Object())
	}
	return rows.Err()
}

func (m *baseManager) Sync(tx *sql.Tx) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.consumer.ConsumeEvents(tx, m.consumeEvent)
}

func (m *baseManager) createObjectEvent(
	tx *sql.Tx, event ObjectEvent,
) (ObjectEvent, error) {
	switch object := event.Object(); event.EventType() {
	case CreateEvent:
		object, err := m.objects.CreateObject(tx, object)
		if err != nil {
			return nil, err
		}
		event = event.WithObject(object)
	case UpdateEvent:
		object, err := m.objects.UpdateObject(tx, object)
		if err != nil {
			return nil, err
		}
		event = event.WithObject(object)
	case DeleteEvent:
		if err := m.objects.DeleteObject(tx, object.ObjectId()); err != nil {
			return nil, err
		}
	}
	result, err := m.events.CreateEvent(tx, event)
	if err != nil {
		return nil, err
	}
	return result.(ObjectEvent), err
}

func (m *baseManager) consumeEvent(e db.Event) error {
	switch v := e.(ObjectEvent); v.EventType() {
	case CreateEvent:
		m.impl.onCreateObject(v.Object())
	case DeleteEvent:
		m.impl.onDeleteObject(v.Object())
	case UpdateEvent:
		m.impl.onUpdateObject(v.Object())
	default:
		return fmt.Errorf("unexpected event type: %v", v.EventType())
	}
	return nil
}

func makeBaseManager(
	object db.Object, table string,
	event ObjectEvent, eventTable string,
	impl baseManagerImpl, dialect db.Dialect,
) baseManager {
	return baseManager{
		objects: db.NewObjectStore(object, "id", table, dialect),
		events:  db.NewEventStore(event, "event_id", eventTable, dialect),
		impl:    impl,
	}
}
