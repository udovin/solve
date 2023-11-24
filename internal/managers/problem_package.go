package managers

import (
	"context"

	"github.com/udovin/solve/internal/pkg/cache"
)

type Problem interface {
}

type problemPackageKey struct {
	ID   int64
	Kind string
}

type ProblemPackageManager struct {
	files *FileManager
	dir   string
	meta  map[string]struct{}
	cache cache.Manager[problemPackageKey, Problem]
}

func NewProblemPackageManager(files *FileManager, dir string) *ProblemPackageManager {
	m := ProblemPackageManager{
		files: files,
		dir:   dir,
		meta:  make(map[string]struct{}),
	}
	m.cache = cache.NewManager[problemPackageKey, Problem](problemPackageManagerStorage{&m})
	return &m
}

func (m *ProblemPackageManager) Download(ctx context.Context, fileID int64, kind string) (cache.Resource[Problem], error) {
	return m.cache.Load(ctx, problemPackageKey{ID: fileID, Kind: kind})
}

func (m *ProblemPackageManager) load(ctx context.Context, fileID int64, kind string) (cache.Resource[Problem], error) {
	file, err := m.files.DownloadFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	panic("not implemented")
}

type problemPackageManagerStorage struct {
	*ProblemPackageManager
}

func (s problemPackageManagerStorage) Load(ctx context.Context, key problemPackageKey) (cache.Resource[Problem], error) {
	return s.ProblemPackageManager.load(ctx, key.ID, key.Kind)
}
