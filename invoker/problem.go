package invoker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/udovin/algo/futures"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

type ProblemTest interface {
}

type ProblemTestGroup interface {
	TimeLimit() int64
	MemoryLimit() int64
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
	GetTestGroups() ([]ProblemTestGroup, error)
	GetStatements() ([]ProblemStatement, error)
}

type ProblemKind string

const (
	PolygonProblem  ProblemKind = "polygon"
	CompiledProblem ProblemKind = "compiled"
)

type problemManager struct {
	cacheDir string
	files    *managers.FileManager
	problems map[int64]futures.Future[Problem]
	mutex    sync.Mutex
}

func newProblemManager(files *managers.FileManager, cacheDir string) (*problemManager, error) {
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return nil, err
	}
	return &problemManager{
		cacheDir: cacheDir,
		files:    files,
		problems: map[int64]futures.Future[Problem]{},
	}, nil
}

func (m *problemManager) DownloadProblem(
	ctx context.Context, packageID int64, kind ProblemKind,
) (Problem, error) {
	return m.downloadProblemAsync(ctx, packageID, kind).Get(ctx)
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
