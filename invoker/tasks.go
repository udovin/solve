package invoker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg/logs"
)

type taskImpl interface {
	New(invoker *Invoker) taskImpl
	Execute(ctx TaskContext) error
}

var registeredTasks = map[models.TaskKind]taskImpl{}

func registerTaskImpl(kind models.TaskKind, impl taskImpl) {
	if _, ok := registeredTasks[kind]; ok {
		panic(fmt.Sprintf("task %q already registered", kind))
	}
	registeredTasks[kind] = impl
}

func isSupportedTask(kind models.TaskKind) bool {
	_, ok := registeredTasks[kind]
	return ok
}

type TaskContext interface {
	context.Context
	Kind() models.TaskKind
	Status() models.TaskStatus
	ScanConfig(models.TaskConfig) error
	ScanState(any) error
	SetStatus(context.Context, models.TaskStatus) error
	SetState(context.Context, any) error
	Ping(context.Context, time.Duration) error
	Logger() *logs.Logger
}

var popTaskMutex sync.Mutex

func popQueuedTask(ctx context.Context, store *models.TaskStore) (*taskGuard, error) {
	popTaskMutex.Lock()
	defer popTaskMutex.Unlock()
	task, err := store.PopQueued(ctx, pingDuration, isSupportedTask)
	if err != nil {
		return nil, err
	}
	guard := &taskGuard{
		store: store,
		task:  task,
	}
	return guard, nil
}

func newTaskContext(ctx context.Context, task *taskGuard, logger *logs.Logger) *taskContext {
	taskCtx, cancel := context.WithCancel(ctx)
	taskPinger := &taskContext{
		taskGuard: task,
		ctx:       taskCtx,
		cancel:    cancel,
		logger:    logger,
	}
	taskPinger.waiter.Add(1)
	go taskPinger.pinger()
	return taskPinger
}

type taskContext struct {
	*taskGuard
	ctx    context.Context
	cancel context.CancelFunc
	waiter sync.WaitGroup
	logger *logs.Logger
}

func (t *taskContext) Close() {
	t.cancel()
	t.waiter.Wait()
}

func (t *taskContext) Done() <-chan struct{} {
	return t.ctx.Done()
}

func (t *taskContext) Deadline() (time.Time, bool) {
	return t.ctx.Deadline()
}

func (t *taskContext) Err() error {
	return t.ctx.Err()
}

func (t *taskContext) Value(key any) any {
	return t.ctx.Value(key)
}

func (t *taskContext) Logger() *logs.Logger {
	return t.logger
}

func (t *taskContext) pinger() {
	defer t.waiter.Done()
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case now := <-ticker.C:
			deadline := t.getDeadline()
			if now.After(deadline) {
				t.Close()
				return
			}
			if now.Add(pingDuration / 2).Before(deadline) {
				continue
			}
			_ = t.Ping(t.ctx, pingDuration)
		case <-t.Done():
			return
		}
	}
}

var _ context.Context = (*taskContext)(nil)

type taskGuard struct {
	store *models.TaskStore
	task  models.Task
	mutex sync.RWMutex
}

const (
	minDuration  = 2 * time.Second
	pingDuration = 10 * minDuration
)

func (t *taskGuard) ObjectID() int64 {
	return t.task.ObjectID()
}

func (t *taskGuard) Kind() models.TaskKind {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.task.Kind
}

func (t *taskGuard) Status() models.TaskStatus {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.task.Status
}

func (t *taskGuard) ScanConfig(config models.TaskConfig) error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.task.ScanConfig(config)
}

func (t *taskGuard) ScanState(state any) error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.task.ScanState(state)
}

func (t *taskGuard) SetStatus(ctx context.Context, status models.TaskStatus) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if err := t.check(); err != nil {
		return err
	}
	clone := t.task.Clone()
	clone.Status = status
	return t.update(ctx, clone)
}

func (t *taskGuard) SetState(ctx context.Context, state any) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if err := t.check(); err != nil {
		return err
	}
	clone := t.task.Clone()
	if err := clone.SetState(state); err != nil {
		return err
	}
	return t.update(ctx, clone)
}

func (t *taskGuard) Ping(ctx context.Context, duration time.Duration) error {
	if duration < minDuration {
		duration = minDuration
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if err := t.check(); err != nil {
		return err
	}
	clone := t.task.Clone()
	clone.ExpireTime = models.NInt64(time.Now().Add(duration).Unix())
	return t.update(ctx, clone)
}

func (t *taskGuard) check() error {
	if t.task.Status != models.RunningTask {
		return fmt.Errorf("task is not running")
	}
	if time.Now().Unix() >= int64(t.task.ExpireTime) {
		return fmt.Errorf("task is expired")
	}
	return nil
}

func (t *taskGuard) update(ctx context.Context, task models.Task) error {
	updateCtx, cancel := context.WithDeadline(ctx, time.Unix(int64(t.task.ExpireTime), 0))
	defer cancel()
	if err := t.store.Update(updateCtx, task); err != nil {
		return err
	}
	t.task = task
	return nil
}

func (t *taskGuard) getDeadline() time.Time {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return time.Unix(int64(t.task.ExpireTime), 0)
}
