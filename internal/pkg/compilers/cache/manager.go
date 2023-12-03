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
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/archives"
	"github.com/udovin/solve/internal/pkg/cache"
	"github.com/udovin/solve/internal/pkg/compilers"
	"github.com/udovin/solve/internal/pkg/safeexec"
)

type CompilerImage interface {
	Compiler(name string, config models.CompilerConfig) compilers.Compiler
}

type compilerImage struct {
	id  int64
	mgr *CompilerImageManager
}

func (r *compilerImage) Compiler(name string, config models.CompilerConfig) compilers.Compiler {
	return &compiler{
		safeexec: r.mgr.safeexec,
		layer:    filepath.Join(r.mgr.dir, fmt.Sprint(r.id)),
		name:     name,
		config:   config,
	}
}

func (r *compilerImage) Get() CompilerImage {
	return r
}

func (r *compilerImage) Release() {
	r.mgr.deleteImage(r)
}

type CompilerImageManager struct {
	files    *managers.FileManager
	safeexec *safeexec.Manager
	dir      string
	images   map[int64]*compilerImage
	seqID    atomic.Int64
	mutex    sync.Mutex
	cache    cache.Manager[int64, CompilerImage]
}

func NewCompilerImageManager(files *managers.FileManager, safeexec *safeexec.Manager, dir string) *CompilerImageManager {
	m := CompilerImageManager{
		files:    files,
		safeexec: safeexec,
		dir:      dir,
		images:   map[int64]*compilerImage{},
	}
	m.cache = cache.NewManager[int64, CompilerImage](compilerImageManagerStorage{&m})
	return &m
}

func (m *CompilerImageManager) LoadSync(
	ctx context.Context, fileID int64,
) (cache.Resource[CompilerImage], error) {
	return m.cache.LoadSync(ctx, fileID)
}

func (m *CompilerImageManager) load(
	ctx context.Context, fileID int64,
) (cache.Resource[CompilerImage], error) {
	file, err := m.files.DownloadFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	img, err := m.newImage()
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			img.Release()
		}
	}()
	targetPath := filepath.Join(m.dir, fmt.Sprint(img.id))
	if err := os.RemoveAll(targetPath); err != nil {
		return nil, err
	}
	tempPath := filepath.Join(m.dir, fmt.Sprintf("%d.tmp", img.id))
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
	if err := archives.ExtractTarGz(tempPath, targetPath); err != nil {
		return nil, fmt.Errorf("cannot extract image: %w", err)
	}
	success = true
	return img, nil
}

func (m *CompilerImageManager) newImage() (*compilerImage, error) {
	id := m.seqID.Add(1)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	p := &compilerImage{id: id, mgr: m}
	m.images[id] = p
	return p, nil
}

func (m *CompilerImageManager) deleteImage(r *compilerImage) {
	// Delete all image files.
	_ = os.RemoveAll(filepath.Join(m.dir, fmt.Sprint(r.id)))
	// Delete information about image.
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.images, r.id)
}

type compilerImageManagerStorage struct {
	*CompilerImageManager
}

func (s compilerImageManagerStorage) Load(
	ctx context.Context, key int64,
) (cache.Resource[CompilerImage], error) {
	return s.CompilerImageManager.load(ctx, key)
}
