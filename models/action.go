package models

import (
	"database/sql"
	"time"

	"github.com/udovin/solve/db"
)

type BaseEvent struct {
	BaseEventID   int64 `db:"event_id" json:"EventID"`
	BaseEventTime int64 `db:"event_time" json:"EventTime"`
}

func (e BaseEvent) EventID() int64 {
	return e.BaseEventID
}

func (e BaseEvent) EventTime() time.Time {
	return time.Unix(e.BaseEventTime, 0)
}

type baseManagerImpl interface {
	clear()
	addObject(db.Object) error
	onObjectEvent(db.Event) error
}

type baseManager struct {
	objects  db.ObjectStore
	events   db.EventStore
	consumer db.EventConsumer
	impl     baseManagerImpl
}

func (m *baseManager) Init(tx *sql.Tx) error {
	rows, err := m.objects.LoadObjects(tx)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()
	m.impl.clear()
	m.consumer = db.NewEventConsumer(m.events, 1)
	for rows.Next() {
		if err := m.impl.addObject(rows.Object()); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (m *baseManager) Sync(tx *sql.Tx) error {
	return m.consumer.ConsumeEvents(tx, m.impl.onObjectEvent)
}

func makeBaseManager(
	object db.Object, table string,
	event db.Event, eventTable string,
	impl baseManagerImpl, dialect db.Dialect,
) baseManager {
	return baseManager{
		objects: db.NewObjectStore(object, "id", table, dialect),
		events:  db.NewEventStore(event, "event_id", eventTable, dialect),
		impl:    impl,
	}
}

type ActionType int

type Action struct {
	ID   int64      `db:"id"`
	Type ActionType `db:"type"`
	Data string     `db:"data"`
}

func (o Action) ObjectID() int64 {
	return o.ID
}

type ActionEvent struct {
	BaseEvent
	Action
}

type ActionManager struct {
	baseManager
}

func (m *ActionManager) clear() {
	panic("implement me")
}

func (m *ActionManager) addObject(db.Object) error {
	panic("implement me")
}

func (m *ActionManager) onObjectEvent(db.Event) error {
	panic("implement me")
}

func NewActionManager(
	table, eventTable string, dialect db.Dialect,
) *ActionManager {
	impl := &ActionManager{}
	impl.baseManager = makeBaseManager(
		Action{}, table, ActionEvent{}, eventTable, impl, dialect,
	)
	return impl
}
