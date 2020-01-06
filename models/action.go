package models

import (
	"database/sql"
	"fmt"

	"github.com/udovin/solve/db"
)

// ActionStatus represents status of action.
type ActionStatus int

const (
	// Queued means that action in queue and should be processed.
	Queued ActionStatus = 0
	// Running means that action already in processing.
	Running ActionStatus = 1
	// Succeeded means that action is processed with success.
	Succeeded ActionStatus = 2
	// Failed means that action is processed with failure.
	Failed ActionStatus = 3
)

// String returns string representation.
func (t ActionStatus) String() string {
	switch t {
	case Queued:
		return "Queued"
	case Running:
		return "Running"
	case Succeeded:
		return "Succeeded"
	case Failed:
		return "Failed"
	default:
		return fmt.Sprintf("ActionStatus(%d)", t)
	}
}

// MarshalText marshals status to text.
func (t ActionStatus) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// ActionType represents type of action.
type ActionType int

const (
	// JudgeSolution represents judge solution action.
	JudgeSolution ActionType = 1
)

// String returns string representation.
func (t ActionType) String() string {
	switch t {
	case JudgeSolution:
		return "JudgeSolution"
	default:
		return fmt.Sprintf("ActionType(%d)", t)
	}
}

// MarshalText marshals type to text.
func (t ActionType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// Action represents action.
type Action struct {
	ID         int64        `db:"id"`
	Status     ActionStatus `db:"status"`
	Type       ActionType   `db:"type"`
	Config     []byte       `db:"config"`
	State      []byte       `db:"state"`
	ExpireTime int64        `db:"expire_time"`
}

// ObjectId returns ID of action.
func (o Action) ObjectID() int64 {
	return o.ID
}

// ActionEvent represents action event.
type ActionEvent struct {
	baseEvent
	Action
}

// Object returns action.
func (e ActionEvent) Object() db.Object {
	return e.Action
}

// WithObject returns action event with specified object.
func (e ActionEvent) WithObject(o db.Object) ObjectEvent {
	e.Action = o.(Action)
	return e
}

// ActionManager represents manager for actions.
type ActionManager struct {
	baseManager
	actions  map[int64]Action
	byStatus indexInt64
}

// Get returns action by id.
//
// Returns sql.ErrNoRows if action does not exist.
func (m *ActionManager) Get(id int64) (Action, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if action, ok := m.actions[id]; ok {
		return action, nil
	}
	return Action{}, sql.ErrNoRows
}

// FindByStatus returns a list of actions with specified status.
func (m *ActionManager) FindByStatus(status ActionStatus) ([]Action, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var actions []Action
	for id := range m.byStatus[int64(status)] {
		if action, ok := m.actions[id]; ok {
			actions = append(actions, action)
		}
	}
	return actions, nil
}

// CreateTx creates action and returns copy with valid ID.
func (m *ActionManager) CreateTx(tx *sql.Tx, action Action) (Action, error) {
	event, err := m.createObjectEvent(tx, ActionEvent{
		makeBaseEvent(CreateEvent),
		action,
	})
	if err != nil {
		return Action{}, err
	}
	return event.Object().(Action), nil
}

// UpdateTx updates action.
func (m *ActionManager) UpdateTx(tx *sql.Tx, action Action) error {
	_, err := m.createObjectEvent(tx, ActionEvent{
		makeBaseEvent(UpdateEvent),
		action,
	})
	return err
}

// DeleteTx deletes action.
func (m *ActionManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, ActionEvent{
		makeBaseEvent(DeleteEvent),
		Action{ID: id},
	})
	return err
}

// PopQueuedTx pops queued action from the store and sets running status.
func (m *ActionManager) PopQueuedTx(tx *sql.Tx) (Action, error) {
	if err := m.lockStore(tx); err != nil {
		return Action{}, err
	}
	if err := m.SyncTx(tx); err != nil {
		return Action{}, err
	}
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for id := range m.byStatus[int64(Queued)] {
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
	m.byStatus = indexInt64{}
}

func (m *ActionManager) addObject(o db.Object) {
	m.onCreateObject(o)
}

func (m *ActionManager) onCreateObject(o db.Object) {
	action := o.(Action)
	m.actions[action.ID] = action
	m.byStatus.Create(int64(action.Status), action.ID)
}

func (m *ActionManager) onDeleteObject(o db.Object) {
	action := o.(Action)
	m.byStatus.Delete(int64(action.Status), action.ID)
	delete(m.actions, action.ID)
}

func (m *ActionManager) onUpdateObject(o db.Object) {
	action := o.(Action)
	if old, ok := m.actions[action.ID]; ok {
		if old.Status != action.Status {
			m.byStatus.Delete(int64(old.Status), old.ID)
		}
	}
	m.onCreateObject(o)
}

// NewActionManager creates a new instance of ActionManager.
func NewActionManager(
	table, eventTable string, dialect db.Dialect,
) *ActionManager {
	impl := &ActionManager{}
	impl.baseManager = makeBaseManager(
		Action{}, table, ActionEvent{}, eventTable, impl, dialect,
	)
	return impl
}
