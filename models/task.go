package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/udovin/gosql"
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

// JudgeSolutionConfig represets config for JudgeSolution.
type JudgeSolutionConfig struct {
	SolutionID int64 `json:"solution_id"`
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

// Clone create copy of task.
func (o Task) Clone() Task {
	o.Config = o.Config.Clone()
	o.State = o.State.Clone()
	return o
}

func (o Task) ScanConfig(config any) error {
	return json.Unmarshal(o.Config, config)
}

func (o *Task) SetConfig(config any) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	o.Config = raw
	return nil
}

func (o Task) ScanState(state any) error {
	return json.Unmarshal(o.Config, state)
}

func (o *Task) SetState(state any) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	o.Config = raw
	return nil
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
		return task.Clone(), nil
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
			tasks = append(tasks, task.Clone())
		}
	}
	return tasks, nil
}

// CreateTx creates task and returns copy with valid ID.
func (s *TaskStore) CreateTx(tx gosql.WeakTx, task *Task) error {
	event, err := s.createObjectEvent(tx, TaskEvent{
		makeBaseEvent(CreateEvent),
		*task,
	})
	if err != nil {
		return err
	}
	*task = event.Object().(Task)
	return nil
}

// UpdateTx updates task.
func (s *TaskStore) UpdateTx(tx gosql.WeakTx, task Task) error {
	_, err := s.createObjectEvent(tx, TaskEvent{
		makeBaseEvent(UpdateEvent),
		task,
	})
	return err
}

// DeleteTx deletes action.
func (s *TaskStore) DeleteTx(tx gosql.WeakTx, id int64) error {
	_, err := s.createObjectEvent(tx, TaskEvent{
		makeBaseEvent(DeleteEvent),
		Task{ID: id},
	})
	return err
}

// PopQueuedTx pops queued action from the events and sets running status.
//
// Note that events is not synchronized after tasks is popped.
func (s *TaskStore) PopQueuedTx(tx gosql.WeakTx) (Task, error) {
	var task Task
	if err := gosql.WithEnsuredTx(tx, func(tx *sql.Tx) (err error) {
		task, err = s.popQueuedTx(tx)
		return
	}); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *TaskStore) popQueuedTx(tx *sql.Tx) (Task, error) {
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
			task = task.Clone()
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
	db *gosql.DB, table, eventTable string,
) *TaskStore {
	impl := &TaskStore{}
	impl.baseStore = makeBaseStore(
		db, Task{}, table, TaskEvent{}, eventTable, impl,
	)
	return impl
}
