package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

// ActionStatus
type ActionStatus int

const (
	Queued    ActionStatus = 0
	Running   ActionStatus = 1
	Succeeded ActionStatus = 2
	Failed    ActionStatus = 3
)

// ActionType represents type of action
type ActionType int

const (
	JudgeSolution ActionType = 1
)

// Action represents action
type Action struct {
	Id      int64        `db:"id"`
	Status  ActionStatus `db:"status"`
	Type    ActionType   `db:"type"`
	Config  []byte       `db:"config"`
	State   []byte       `db:"state"`
	EndTime int64        `db:"end_time"`
}

// ObjectId returns id of action
func (o Action) ObjectId() int64 {
	return o.Id
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
	actions  map[int64]Action
	byStatus map[ActionStatus]map[int64]struct{}
}

func (m *ActionManager) Get(id int64) (Action, error) {
	if action, ok := m.actions[id]; ok {
		return action, nil
	}
	return Action{}, sql.ErrNoRows
}

func (m *ActionManager) CreateTx(tx *sql.Tx, action *Action) error {
	event, err := m.createObjectEvent(tx, ActionEvent{
		makeBaseEvent(CreateEvent),
		*action,
	})
	if err != nil {
		return err
	}
	*action = event.Object().(Action)
	return nil
}

func (m *ActionManager) UpdateTx(tx *sql.Tx, action Action) error {
	_, err := m.createObjectEvent(tx, ActionEvent{
		makeBaseEvent(UpdateEvent),
		action,
	})
	return err
}

func (m *ActionManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, ActionEvent{
		makeBaseEvent(DeleteEvent),
		Action{Id: id},
	})
	return err
}

func (m *ActionManager) PopQueuedTx(tx *sql.Tx) (Action, error) {
	if err := m.Sync(tx); err != nil {
		return Action{}, err
	}
	for id := range m.byStatus[Queued] {
		if action, ok := m.actions[id]; ok {
			action.Status = Running
			if err := m.UpdateTx(tx, action); err != nil {
				return Action{}, err
			}
			return action, nil
		}
	}
	return Action{}, sql.ErrNoRows
}

func (m *ActionManager) reset() {
	m.actions = map[int64]Action{}
	m.byStatus = map[ActionStatus]map[int64]struct{}{}
}

func (m *ActionManager) addObject(o db.Object) {
	m.onCreateObject(o)
}

func (m *ActionManager) onCreateObject(o db.Object) {
	action := o.(Action)
	m.actions[action.Id] = action
	if _, ok := m.byStatus[action.Status]; !ok {
		m.byStatus[action.Status] = map[int64]struct{}{}
	}
	m.byStatus[action.Status][action.Id] = struct{}{}
}

func (m *ActionManager) onDeleteObject(o db.Object) {
	action := o.(Action)
	delete(m.byStatus[action.Status], action.Id)
	if len(m.byStatus[action.Status]) == 0 {
		delete(m.byStatus, action.Status)
	}
	delete(m.actions, o.ObjectId())
}

func (m *ActionManager) onUpdateObject(o db.Object) {
	action := o.(Action)
	if old, ok := m.actions[action.Id]; ok {
		if old.Status != action.Status {
			delete(m.byStatus[old.Status], action.Id)
			if len(m.byStatus[old.Status]) == 0 {
				delete(m.byStatus, old.Status)
			}
		}
	}
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
