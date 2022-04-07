package invoker

import (
	"context"
	"fmt"

	"github.com/udovin/solve/models"
)

type taskImpl interface {
	New(invoker Invoker) taskImpl
	Execute(ctx context.Context, setState func(state models.JSON) error) error
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
