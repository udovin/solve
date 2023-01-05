package invoker

import (
	"context"
	"fmt"

	"github.com/udovin/solve/pkg"
)

func extractCompiledProblem(
	source, target string, compilers *compilerManager,
) (Problem, error) {
	if err := pkg.ExtractTarGz(source, target); err != nil {
		return nil, fmt.Errorf("cannot extract problem: %w", err)
	}
	problem := compiledProblem{
		path:      target,
		compilers: compilers,
	}
	return &problem, nil
}

type compiledProblem struct {
	path      string
	compilers *compilerManager
}

func (p *compiledProblem) Compile(ctx context.Context) error {
	return nil
}

func (p *compiledProblem) GetTestGroups() ([]ProblemTestGroup, error) {
	return nil, nil
}

func (p *compiledProblem) GetStatements() ([]ProblemStatement, error) {
	return nil, nil
}

var _ Problem = (*compiledProblem)(nil)
