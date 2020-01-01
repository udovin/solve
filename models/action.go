package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

// ActionType represents type of action
type ActionType int

// Action represents action
type Action struct {
	ID   int64      `db:"id"`
	Type ActionType `db:"type"`
	Data string     `db:"data"`
}

// ObjectID returns ID of action
func (o Action) ObjectID() int64 {
	return o.ID
}

// ActionEvent represents action event
type ActionEvent struct {
	baseEvent
	Action
}

// Object returns action
func (e ActionEvent) Object() db.Object {
	return e.Action
}

// WithObject returns action event with specified object
func (e ActionEvent) WithObject(o db.Object) ObjectEvent {
	e.Action = o.(Action)
	return e
}

// ActionManager represents manager for actions
type ActionManager struct {
	baseManager
	actions map[int64]Action
}

func (m *ActionManager) reset() {
	m.actions = map[int64]Action{}
}

func (m *ActionManager) addObject(o db.Object) {
	m.onCreateObject(o)
}

func (m *ActionManager) onCreateObject(o db.Object) {
	m.actions[o.ObjectID()] = o.(Action)
}

func (m *ActionManager) onDeleteObject(o db.Object) {
	delete(m.actions, o.ObjectID())
}

func (m *ActionManager) onUpdateObject(o db.Object) {
	m.onCreateObject(o)
}

func (m *ActionManager) updateSchema(tx *sql.Tx, version int) (int, error) {
	panic("implement me")
}

// NewActionManager creates a new instance of ActionManager
func NewActionManager(
	table, eventTable string, dialect db.Dialect,
) *ActionManager {
	impl := &ActionManager{}
	impl.baseManager = makeBaseManager(
		Action{}, table, ActionEvent{}, eventTable, impl, dialect,
	)
	return impl
}
