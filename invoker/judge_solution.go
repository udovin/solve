package invoker

import (
	"fmt"

	"github.com/udovin/solve/models"
)

func init() {
	registerTaskImpl(models.JudgeSolutionTask, &judgeSolutionTask{})
}

type judgeSolutionTask struct {
}

func (judgeSolutionTask) New(invoker *Invoker) taskImpl {
	return &judgeSolutionTask{}
}

func (t *judgeSolutionTask) Execute(ctx TaskContext) error {
	return fmt.Errorf("not implemented")
}
