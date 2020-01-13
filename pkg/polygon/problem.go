package polygon

import (
	"github.com/udovin/solve/pkg"
)

type problem struct{}

func (p *problem) Statements() ([]pkg.Statement, error) {
	panic("implement me")
}

func (p *problem) TestGroups() ([]pkg.TestGroup, error) {
	panic("implement me")
}

// ReadProblem reads problem from directory.
func ReadProblem(dir string) (pkg.Problem, error) {
	return &problem{}, nil
}
