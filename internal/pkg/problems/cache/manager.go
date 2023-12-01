package cache

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/pkg/cache"
	"github.com/udovin/solve/internal/pkg/problems"
)

type problemPackageKey struct {
	ID   int64
	Kind string
}

type problemPackage struct {
	id      int64
	mgr     *ProblemPackageManager
	problem problems.Problem
}

func (p *problemPackage) Get() problems.Problem {
	return p.problem
}

func (p *problemPackage) Release() {
	p.mgr.deletePackage(p)
}

type ProblemPackageManager struct {
	files    *managers.FileManager
	dir      string
	packages map[int64]*problemPackage
	seqID    atomic.Int64
	mutex    sync.Mutex
	cache    cache.Manager[problemPackageKey, problems.Problem]
}

func NewProblemPackageManager(files *managers.FileManager, dir string) *ProblemPackageManager {
	m := ProblemPackageManager{
		files:    files,
		dir:      dir,
		packages: map[int64]*problemPackage{},
	}
	m.cache = cache.NewManager[problemPackageKey, problems.Problem](problemPackageManagerStorage{&m})
	return &m
}

func (m *ProblemPackageManager) Download(
	ctx context.Context, fileID int64, kind string,
) cache.ResourceFuture[problems.Problem] {
	return m.cache.Load(ctx, problemPackageKey{ID: fileID, Kind: kind})
}

func (m *ProblemPackageManager) load(
	ctx context.Context, fileID int64, kind string,
) (cache.Resource[problems.Problem], error) {
	file, err := m.files.DownloadFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	pkg, err := m.newPackage()
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			pkg.Release()
		}
	}()
	targetPath := filepath.Join(m.dir, fmt.Sprint(pkg.id))
	if err := os.RemoveAll(targetPath); err != nil {
		return nil, err
	}
	tempPath := filepath.Join(m.dir, fmt.Sprintf("%d.tmp", pkg.id))
	if err := os.RemoveAll(tempPath); err != nil {
		return nil, err
	}
	if v, ok := file.(*os.File); ok {
		tempPath = v.Name()
	} else {
		tempFile, err := os.Create(tempPath)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = tempFile.Close()
			_ = os.RemoveAll(tempPath)
		}()
		if _, err := io.Copy(tempFile, file); err != nil {
			return nil, err
		}
		if err := tempFile.Close(); err != nil {
			return nil, err
		}
	}
	problem, err := extractProblem(kind, targetPath, tempPath)
	if err != nil {
		return nil, err
	}
	pkg.problem = problem
	success = true
	return pkg, nil
}

func (m *ProblemPackageManager) newPackage() (*problemPackage, error) {
	id := m.seqID.Add(1)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	p := &problemPackage{id: id, mgr: m}
	m.packages[id] = p
	return p, nil
}

func (m *ProblemPackageManager) deletePackage(p *problemPackage) {
	// Delete all package files.
	_ = os.RemoveAll(filepath.Join(m.dir, fmt.Sprint(p.id)))
	// Delete information about package.
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.packages, p.id)
}

type problemPackageManagerStorage struct {
	*ProblemPackageManager
}

func (s problemPackageManagerStorage) Load(
	ctx context.Context, key problemPackageKey,
) (cache.Resource[problems.Problem], error) {
	return s.ProblemPackageManager.load(ctx, key.ID, key.Kind)
}

func extractProblem(
	kind string, targetPath, sourcePath string,
) (problems.Problem, error) {
	panic("not implemented")
}
