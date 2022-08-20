package models

import (
	"context"
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
	// QueuedTask means that task in queue and should be processed.
	QueuedTask TaskStatus = 0
	// RunningTask means that task already in processing.
	RunningTask TaskStatus = 1
	// SucceededTask means that task is processed with success.
	SucceededTask TaskStatus = 2
	// FailedTask means that task is processed with failure.
	FailedTask TaskStatus = 3
)

// String returns string representation.
func (t TaskStatus) String() string {
	switch t {
	case QueuedTask:
		return "queued"
	case RunningTask:
		return "running"
	case SucceededTask:
		return "succeeded"
	case FailedTask:
		return "failed"
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
	JudgeSolutionTask TaskKind = 1
)

// String returns string representation.
func (t TaskKind) String() string {
	switch t {
	case JudgeSolutionTask:
		return "judge_solution"
	default:
		return fmt.Sprintf("TaskKind(%d)", t)
	}
}

// MarshalText marshals kind to text.
func (t TaskKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// JudgeSolutionTaskConfig represets config for JudgeSolution.
type JudgeSolutionTaskConfig struct {
	SolutionID int64 `json:"solution_id"`
}

func (c JudgeSolutionTaskConfig) TaskKind() TaskKind {
	return JudgeSolutionTask
}

type TaskConfig interface {
	TaskKind() TaskKind
}

// Task represents async task.
type Task struct {
	baseObject
	Kind       TaskKind   `db:"kind"`
	Config     JSON       `db:"config"`
	Status     TaskStatus `db:"status"`
	State      JSON       `db:"state"`
	ExpireTime int64      `db:"expire_time"`
}

// Clone create copy of task.
func (o Task) Clone() Task {
	o.Config = o.Config.Clone()
	o.State = o.State.Clone()
	return o
}

func (o Task) ScanConfig(config TaskConfig) error {
	return json.Unmarshal(o.Config, config)
}

// SetConfig updates kind and config of task.
func (o *Task) SetConfig(config TaskConfig) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	o.Kind = config.TaskKind()
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
func (e TaskEvent) Object() Task {
	return e.Task
}

// SetObject sets event task.
func (e *TaskEvent) SetObject(o Task) {
	e.Task = o
}

// TaskStore represents store for tasks.
type TaskStore struct {
	baseStore[Task, TaskEvent, *Task, *TaskEvent]
	tasks    map[int64]Task
	byStatus index[TaskStatus]
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
	for id := range s.byStatus[status] {
		if task, ok := s.tasks[id]; ok {
			tasks = append(tasks, task.Clone())
		}
	}
	return tasks, nil
}

// PopQueued pops queued action from the events and sets running status.
//
// Note that events is not synchronized after tasks is popped.
func (s *TaskStore) PopQueued(
	ctx context.Context,
	filter func(TaskKind) bool,
) (Task, error) {
	tx := db.GetTx(ctx)
	if tx == nil {
		var task Task
		err := gosql.WrapTx(ctx, s.db, func(tx *sql.Tx) (err error) {
			task, err = s.PopQueued(db.WithTx(ctx, tx), filter)
			return err
		}, sqlRepeatableRead)
		return task, err
	}
	if err := s.lockStore(tx); err != nil {
		return Task{}, err
	}
	if err := s.Sync(ctx); err != nil {
		return Task{}, err
	}
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byStatus[QueuedTask] {
		if task, ok := s.tasks[id]; ok && filter(task.Kind) {
			// We should make clone of action, because we do not
			// want to corrupt Store in-memory cache.
			task = task.Clone()
			// Now we can do any manipulations with this action.
			task.Status = RunningTask
			task.ExpireTime = time.Now().Add(5 * time.Second).Unix()
			if err := s.Update(ctx, task); err != nil {
				return Task{}, err
			}
			return task, nil
		}
	}
	return Task{}, sql.ErrNoRows
}

func (s *TaskStore) reset() {
	s.tasks = map[int64]Task{}
	s.byStatus = index[TaskStatus]{}
}

func (s *TaskStore) onCreateObject(task Task) {
	s.tasks[task.ID] = task
	s.byStatus.Create(task.Status, task.ID)
}

func (s *TaskStore) onDeleteObject(id int64) {
	if task, ok := s.tasks[id]; ok {
		s.byStatus.Delete(task.Status, task.ID)
		delete(s.tasks, task.ID)
	}
}

var _ baseStoreImpl[Task] = (*TaskStore)(nil)

// NewTaskStore creates a new instance of TaskStore.
func NewTaskStore(
	db *gosql.DB, table, eventTable string,
) *TaskStore {
	impl := &TaskStore{}
	impl.baseStore = makeBaseStore[Task, TaskEvent](
		db, table, eventTable, impl,
	)
	return impl
}
