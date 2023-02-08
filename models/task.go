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
	// JudgeSolutionTask represents judge solution task.
	JudgeSolutionTask TaskKind = 1
	// UpdateProblemPackageTask represents task for update problem package.
	UpdateProblemPackageTask TaskKind = 2
)

// String returns string representation.
func (t TaskKind) String() string {
	switch t {
	case JudgeSolutionTask:
		return "judge_solution"
	case UpdateProblemPackageTask:
		return "update_problem_package"
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

type JudgeSolutionTaskState struct {
	Stage string `json:"stage,omitempty"`
}

// UpdateProblemPackageTaskConfig represets config for JudgeSolution.
type UpdateProblemPackageTaskConfig struct {
	ProblemID int64 `json:"problem_id"`
	FileID    int64 `json:"file_id"`
	Compile   bool  `json:"compile"`
}

func (c UpdateProblemPackageTaskConfig) TaskKind() TaskKind {
	return UpdateProblemPackageTask
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
	ExpireTime NInt64     `db:"expire_time"`
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
	return json.Unmarshal(o.State, state)
}

func (o *Task) SetState(state any) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	o.State = raw
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
	cachedStore[Task, TaskEvent, *Task, *TaskEvent]
	bySolution *index[int64, Task, *Task]
}

// FindBySolution returns a list of tasks by specified solution.
func (s *TaskStore) FindBySolution(id int64) ([]Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []Task
	for id := range s.bySolution.Get(id) {
		if object, ok := s.objects.Get(id); ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

// PopQueued pops queued action from the events and sets running status.
//
// Note that events is not synchronized after tasks is popped.
func (s *TaskStore) PopQueued(
	ctx context.Context,
	duration time.Duration,
	filter func(TaskKind) bool,
) (Task, error) {
	tx := db.GetTx(ctx)
	if tx == nil {
		var task Task
		err := gosql.WrapTx(ctx, s.db, func(tx *sql.Tx) (err error) {
			task, err = s.PopQueued(db.WithTx(ctx, tx), duration, filter)
			return err
		}, sqlRepeatableRead)
		return task, err
	}
	if err := s.lockStore(tx); err != nil {
		return Task{}, err
	}
	reader, err := s.Find(ctx, gosql.Column("status").Equal(QueuedTask))
	if err != nil {
		return Task{}, err
	}
	defer reader.Close()
	for reader.Next() {
		task := reader.Row()
		if filter != nil && !filter(task.Kind) {
			continue
		}
		if task.Status != QueuedTask {
			return Task{}, fmt.Errorf("unexpected status: %s", task.Status)
		}
		if err := reader.Close(); err != nil {
			return Task{}, err
		}
		task.Status = RunningTask
		task.ExpireTime = NInt64(time.Now().Add(duration).Unix())
		if err := s.Update(ctx, task); err != nil {
			return Task{}, err
		}
		return task, nil
	}
	return Task{}, sql.ErrNoRows
}

var _ baseStoreImpl[Task] = (*TaskStore)(nil)

// NewTaskStore creates a new instance of TaskStore.
func NewTaskStore(
	db *gosql.DB, table, eventTable string,
) *TaskStore {
	impl := &TaskStore{
		bySolution: newIndex(func(o Task) int64 {
			switch o.Kind {
			case JudgeSolutionTask:
				var config JudgeSolutionTaskConfig
				if err := o.ScanConfig(&config); err == nil {
					return config.SolutionID
				}
			}
			return 0
		}),
	}
	impl.cachedStore = makeBaseStore[Task, TaskEvent](
		db, table, eventTable, impl, impl.bySolution,
	)
	return impl
}
