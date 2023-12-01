package problems

import (
	"context"
	"os"

	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/compilers"
	"github.com/udovin/solve/internal/pkg/logs"
)

type ProblemKind string

const (
	PolygonProblem  ProblemKind = "polygon"
	CompiledProblem ProblemKind = "compiled"
)

type ProblemTest interface {
	OpenInput() (*os.File, error)
	OpenAnswer() (*os.File, error)
	Points() float64
	Group() string
}

type ProblemExecutableKind string

const (
	TestlibChecker    ProblemExecutableKind = "testlib_checker"
	TestlibInteractor ProblemExecutableKind = "testlib_interactor"
)

type ProblemExecutable interface {
	Name() string
	Kind() ProblemExecutableKind
	OpenBinary() (*os.File, error)
	GetCompiler(context.Context, CompileContext) (compilers.Compiler, error)
}

type ProblemPointsPolicy string

const (
	EachTestPointsPolicy      ProblemPointsPolicy = "each_test"
	CompleteGroupPointsPolicy ProblemPointsPolicy = "complete_group"
)

type ProblemTestGroup interface {
	Name() string
	PointsPolicy() ProblemPointsPolicy
}

type ProblemTestSet interface {
	Name() string
	TimeLimit() int64
	MemoryLimit() int64
	GetTests() ([]ProblemTest, error)
	GetGroups() ([]ProblemTestGroup, error)
}

type ProblemResource interface {
	Name() string
	Open() (*os.File, error)
	GetMD5() (string, error)
}

type ProblemStatement interface {
	Locale() string
	GetConfig() (models.ProblemStatementConfig, error)
	GetResources() ([]ProblemResource, error)
}

type CompileContext interface {
	GetCompiler(ctx context.Context, name string) (compilers.Compiler, error)
	Logger() *logs.Logger
	// deprecated.
	GetCompilerName(name string) (string, error)
}

type Problem interface {
	Compile(context.Context, CompileContext) error
	GetExecutables() ([]ProblemExecutable, error)
	GetTestSets() ([]ProblemTestSet, error)
	GetStatements() ([]ProblemStatement, error)
}
