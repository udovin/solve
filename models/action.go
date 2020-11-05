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
	Config     JSON         `db:"config"`
	State      JSON         `db:"state"`
	ExpireTime int64        `db:"expire_time"`
}

// ObjectID returns ID of action.
func (o Action) ObjectID() int64 {
	return o.ID
}

func (o Action) clone() Action {
	o.Config = o.Config.clone()
	o.State = o.State.clone()
	return o
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

// ActionStore represents store for actions.
type ActionStore struct {
	baseStore
	actions  map[int64]Action
	byStatus indexInt64
}

// Get returns action by id.
//
// Returns sql.ErrNoRows if action does not exist.
func (s *ActionStore) Get(id int64) (Action, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if action, ok := s.actions[id]; ok {
		return action.clone(), nil
	}
	return Action{}, sql.ErrNoRows
}

// FindByStatus returns a list of actions with specified status.
func (s *ActionStore) FindByStatus(status ActionStatus) ([]Action, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var actions []Action
	for id := range s.byStatus[int64(status)] {
		if action, ok := s.actions[id]; ok {
			actions = append(actions, action.clone())
		}
	}
	return actions, nil
}

// CreateTx creates action and returns copy with valid ID.
func (s *ActionStore) CreateTx(tx *sql.Tx, action Action) (Action, error) {
	event, err := s.createObjectEvent(tx, ActionEvent{
		makeBaseEvent(CreateEvent),
		action,
	})
	if err != nil {
		return Action{}, err
	}
	return event.Object().(Action), nil
}

// UpdateTx updates action.
func (s *ActionStore) UpdateTx(tx *sql.Tx, action Action) error {
	_, err := s.createObjectEvent(tx, ActionEvent{
		makeBaseEvent(UpdateEvent),
		action,
	})
	return err
}

// DeleteTx deletes action.
func (s *ActionStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, ActionEvent{
		makeBaseEvent(DeleteEvent),
		Action{ID: id},
	})
	return err
}

// PopQueuedTx pops queued action from the events and sets running status.
//
// Note that events is not synchronized after actions is popped.
func (s *ActionStore) PopQueuedTx(tx *sql.Tx) (Action, error) {
	// First of all we should lock events.
	if err := s.lockStore(tx); err != nil {
		return Action{}, err
	}
	// Now we should load all changes from events.
	if err := s.SyncTx(tx); err != nil {
		return Action{}, err
	}
	// New changes will not be available right now due to locked events.
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byStatus[int64(Queued)] {
		if action, ok := s.actions[id]; ok {
			// We should make clone of action, because we do not
			// want to corrupt Store in-memory cache.
			action = action.clone()
			// Now we can do any manipulations with this action.
			action.Status = Running
			if err := s.UpdateTx(tx, action); err != nil {
				return Action{}, err
			}
			return action, nil
		}
	}
	return Action{}, sql.ErrNoRows
}

func (s *ActionStore) reset() {
	s.actions = map[int64]Action{}
	s.byStatus = indexInt64{}
}

func (s *ActionStore) onCreateObject(o db.Object) {
	action := o.(Action)
	s.actions[action.ID] = action
	s.byStatus.Create(int64(action.Status), action.ID)
}

func (s *ActionStore) onDeleteObject(o db.Object) {
	action := o.(Action)
	s.byStatus.Delete(int64(action.Status), action.ID)
	delete(s.actions, action.ID)
}

func (s *ActionStore) onUpdateObject(o db.Object) {
	action := o.(Action)
	if old, ok := s.actions[action.ID]; ok {
		if old.Status != action.Status {
			s.byStatus.Delete(int64(old.Status), old.ID)
		}
	}
	s.onCreateObject(o)
}

// NewActionStore creates a new instance of ActionStore.
func NewActionStore(
	table, eventTable string, dialect db.Dialect,
) *ActionStore {
	impl := &ActionStore{}
	impl.baseStore = makeBaseStore(
		Action{}, table, ActionEvent{}, eventTable, impl, dialect,
	)
	return impl
}
