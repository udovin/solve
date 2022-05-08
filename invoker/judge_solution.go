package invoker

import (
	"context"
	"fmt"

	"github.com/udovin/solve/models"
)

func init() {
	registerTaskImpl(models.JudgeSolutionTask, &judgeSolutionTask{})
}

type judgeSolutionTask struct {
}

func (judgeSolutionTask) New(invoker Invoker) taskImpl {
	return &judgeSolutionTask{}
}

func (t *judgeSolutionTask) Execute(ctx context.Context, setState func(state models.JSON) error) error {
	return fmt.Errorf("not implemented")
}
