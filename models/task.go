package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/udovin/solve/db"
)

// TaskStatus represents status of task.
type TaskStatus int

const (
	// Queued means that task in queue and should be processed.
	Queued TaskStatus = 0
	// Running means that task already in processing.
	Running TaskStatus = 1
	// Succeeded means that task is processed with success.
	Succeeded TaskStatus = 2
	// Failed means that task is processed with failure.
	Failed TaskStatus = 3
)

// String returns string representation.
func (t TaskStatus) String() string {
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
		return fmt.Sprintf("TaskStatus(%d)", t)
	}
}

// MarshalText marshals status to text.
func (t TaskStatus) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// TaskKind represents kind of task.
type TaskKind int

const (
	// JudgeSolution represents judge solution task.
	JudgeSolution TaskKind = 1
)

// String returns string representation.
func (t TaskKind) String() string {
	switch t {
	case JudgeSolution:
		return "JudgeSolution"
	default:
		return fmt.Sprintf("TaskKind(%d)", t)
	}
}

// MarshalText marshals kind to text.
func (t TaskKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// Task represents async task.
type Task struct {
	ID         int64      `db:"id"`
	Status     TaskStatus `db:"status"`
	Kind       TaskKind   `db:"kind"`
	Config     JSON       `db:"config"`
	State      JSON       `db:"state"`
	ExpireTime int64      `db:"expire_time"`
}

// ObjectID returns ID of task.
func (o Task) ObjectID() int64 {
	return o.ID
}

func (o Task) clone() Task {
	o.Config = o.Config.clone()
	o.State = o.State.clone()
	return o
}

// TaskEvent represents task event.
type TaskEvent struct {
	baseEvent
	Task
}

// Object returns task.
func (e TaskEvent) Object() db.Object {
	return e.Task
}

// WithObject returns task event with specified object.
func (e TaskEvent) WithObject(o db.Object) ObjectEvent {
	e.Task = o.(Task)
	return e
}

// TaskStore represents store for tasks.
type TaskStore struct {
	baseStore
	tasks    map[int64]Task
	byStatus indexInt64
}

// Get returns task by id.
//
// Returns sql.ErrNoRows if task does not exist.
func (s *TaskStore) Get(id int64) (Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if task, ok := s.tasks[id]; ok {
		return task.clone(), nil
	}
	return Task{}, sql.ErrNoRows
}

// FindByStatus returns a list of tasks with specified status.
func (s *TaskStore) FindByStatus(status TaskStatus) ([]Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var tasks []Task
	for id := range s.byStatus[int64(status)] {
		if task, ok := s.tasks[id]; ok {
			tasks = append(tasks, task.clone())
		}
	}
	return tasks, nil
}

// CreateTx creates task and returns copy with valid ID.
func (s *TaskStore) CreateTx(tx *sql.Tx, task Task) (Task, error) {
	event, err := s.createObjectEvent(tx, TaskEvent{
		makeBaseEvent(CreateEvent),
		task,
	})
	if err != nil {
		return Task{}, err
	}
	return event.Object().(Task), nil
}

// UpdateTx updates task.
func (s *TaskStore) UpdateTx(tx *sql.Tx, task Task) error {
	_, err := s.createObjectEvent(tx, TaskEvent{
		makeBaseEvent(UpdateEvent),
		task,
	})
	return err
}

// DeleteTx deletes action.
func (s *TaskStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, TaskEvent{
		makeBaseEvent(DeleteEvent),
		Task{ID: id},
	})
	return err
}

// PopQueuedTx pops queued action from the events and sets running status.
//
// Note that events is not synchronized after tasks is popped.
func (s *TaskStore) PopQueuedTx(tx *sql.Tx) (Task, error) {
	if err := s.lockStore(tx); err != nil {
		return Task{}, err
	}
	if err := s.SyncTx(tx); err != nil {
		return Task{}, err
	}
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byStatus[int64(Queued)] {
		if task, ok := s.tasks[id]; ok {
			// We should make clone of action, because we do not
			// want to corrupt Store in-memory cache.
			task = task.clone()
			// Now we can do any manipulations with this action.
			task.Status = Running
			task.ExpireTime = time.Now().Add(5 * time.Second).Unix()
			if err := s.UpdateTx(tx, task); err != nil {
				return Task{}, err
			}
			return task, nil
		}
	}
	return Task{}, sql.ErrNoRows
}

func (s *TaskStore) reset() {
	s.tasks = map[int64]Task{}
	s.byStatus = indexInt64{}
}

func (s *TaskStore) onCreateObject(o db.Object) {
	task := o.(Task)
	s.tasks[task.ID] = task
	s.byStatus.Create(int64(task.Status), task.ID)
}

func (s *TaskStore) onDeleteObject(o db.Object) {
	task := o.(Task)
	s.byStatus.Delete(int64(task.Status), task.ID)
	delete(s.tasks, task.ID)
}

func (s *TaskStore) onUpdateObject(o db.Object) {
	task := o.(Task)
	if old, ok := s.tasks[task.ID]; ok {
		if old.Status != task.Status {
			s.byStatus.Delete(int64(old.Status), old.ID)
		}
	}
	s.onCreateObject(o)
}

// NewTaskStore creates a new instance of TaskStore.
func NewTaskStore(
	table, eventTable string, dialect db.Dialect,
) *TaskStore {
	impl := &TaskStore{}
	impl.baseStore = makeBaseStore(
		Task{}, table, TaskEvent{}, eventTable, impl, dialect,
	)
	return impl
}
