package invoker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/udovin/algo/futures"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
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
	GetCompiler(context.Context, CompilerManager) (Compiler, error)
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

type Problem interface {
	Compile(context.Context, CompilerManager) error
	GetExecutables() ([]ProblemExecutable, error)
	GetTestSets() ([]ProblemTestSet, error)
	GetStatements() ([]ProblemStatement, error)
}

type ProblemKind string

const (
	PolygonProblem  ProblemKind = "polygon"
	CompiledProblem ProblemKind = "compiled"
)

type problemManager struct {
	files     *managers.FileManager
	cacheDir  string
	problems  map[int64]futures.Future[Problem]
	compilers *compilerManager
	mutex     sync.Mutex
}

func newProblemManager(
	files *managers.FileManager,
	cacheDir string,
	compilers *compilerManager,
) (*problemManager, error) {
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return nil, err
	}
	return &problemManager{
		files:     files,
		cacheDir:  cacheDir,
		problems:  map[int64]futures.Future[Problem]{},
		compilers: compilers,
	}, nil
}

func (m *problemManager) DownloadProblem(
	ctx context.Context, p models.Problem, kind ProblemKind,
) (Problem, error) {
	switch kind {
	case PolygonProblem:
		return m.downloadProblemAsync(ctx, int64(p.PackageID), kind).Get(ctx)
	case CompiledProblem:
		return m.downloadProblemAsync(ctx, int64(p.CompiledID), kind).Get(ctx)
	default:
		return nil, fmt.Errorf("unknown package kind: %v", kind)
	}
}

func (m *problemManager) downloadProblemAsync(
	ctx context.Context, packageID int64, kind ProblemKind,
) futures.Future[Problem] {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if problem, ok := m.problems[packageID]; ok {
		return problem
	}
	future, setResult := futures.New[Problem]()
	m.problems[packageID] = future
	go func() {
		problem, err := m.runDownloadProblem(ctx, packageID, kind)
		if err != nil {
			m.deleteProblem(packageID)
		}
		setResult(problem, err)
	}()
	return future
}

func (m *problemManager) runDownloadProblem(
	ctx context.Context, packageID int64, kind ProblemKind,
) (Problem, error) {
	problemFile, err := m.files.DownloadFile(ctx, packageID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = problemFile.Close() }()
	localProblemPath := filepath.Join(m.cacheDir, fmt.Sprintf("package-%d.zip", packageID))
	_ = os.Remove(localProblemPath)
	problemPath := filepath.Join(m.cacheDir, fmt.Sprintf("package-%d", packageID))
	_ = os.RemoveAll(problemPath)
	if file, ok := problemFile.(*os.File); ok {
		localProblemPath = file.Name()
	} else {
		localProblemFile, err := os.Create(localProblemPath)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = localProblemFile.Close()
			_ = os.Remove(localProblemPath)
		}()
		if _, err := io.Copy(localProblemFile, problemFile); err != nil {
			return nil, err
		}
		if err := localProblemFile.Close(); err != nil {
			return nil, err
		}
	}
	switch kind {
	case PolygonProblem:
		return extractPolygonProblem(localProblemPath, problemPath)
	case CompiledProblem:
		return extractCompiledProblem(localProblemPath, problemPath)
	default:
		return nil, fmt.Errorf("unsupported kind: %v", kind)
	}
}

func (m *problemManager) deleteProblem(packageID int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	problemPath := filepath.Join(m.cacheDir, fmt.Sprintf("package-%d", packageID))
	_ = os.RemoveAll(problemPath)
	delete(m.problems, packageID)
}