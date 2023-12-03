package invoker

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/udovin/algo/futures"
	"github.com/udovin/solve/internal/core"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/archives"
	"github.com/udovin/solve/internal/pkg/compilers"
	"github.com/udovin/solve/internal/pkg/logs"
	"github.com/udovin/solve/internal/pkg/safeexec"
)

type compiler struct {
	safeexec *safeexec.Manager
	name     string
	config   models.CompilerConfig
	layer    string
}

func (c *compiler) Name() string {
	return c.name
}

func (c *compiler) Compile(
	ctx context.Context, options compilers.CompileOptions,
) (compilers.CompileReport, error) {
	if c.config.Compile == nil {
		if err := copyFileRec(options.Source, options.Target); err != nil {
			return compilers.CompileReport{}, fmt.Errorf("unable to copy source: %w", err)
		}
		return compilers.CompileReport{}, nil
	}
	log := truncateBuffer{limit: 2048}
	config := safeexec.ProcessConfig{
		Layers:      []string{c.layer},
		Command:     strings.Fields(c.config.Compile.Command),
		Environ:     c.config.Compile.Environ,
		Workdir:     c.config.Compile.Workdir,
		Stdout:      &log,
		Stderr:      &log,
		TimeLimit:   options.TimeLimit,
		MemoryLimit: options.MemoryLimit,
	}
	process, err := c.safeexec.Create(ctx, config)
	if err != nil {
		return compilers.CompileReport{}, fmt.Errorf("unable to create compiler: %w", err)
	}
	if c.config.Compile.Source != nil {
		path := filepath.Join(
			process.UpperDir(),
			c.config.Compile.Workdir,
			*c.config.Compile.Source,
		)
		if err := copyFileRec(options.Source, path); err != nil {
			return compilers.CompileReport{}, fmt.Errorf("unable to write source: %w", err)
		}
	}
	for _, file := range options.InputFiles {
		path := filepath.Join(
			process.UpperDir(),
			c.config.Compile.Workdir,
			file.Target,
		)
		if err := copyFileRec(file.Source, path); err != nil {
			return compilers.CompileReport{}, fmt.Errorf("unable to write file: %w", err)
		}
	}
	defer func() { _ = process.Release() }()
	if err := process.Start(); err != nil {
		return compilers.CompileReport{}, fmt.Errorf("cannot start compiler: %w", err)
	}
	report, err := process.Wait()
	if err != nil {
		return compilers.CompileReport{}, err
	}
	if report.ExitCode == 0 {
		if c.config.Compile.Binary != nil {
			containerBinaryPath := filepath.Join(
				process.UpperDir(),
				c.config.Compile.Workdir,
				*c.config.Compile.Binary,
			)
			if err := copyFileRec(containerBinaryPath, options.Target); err != nil {
				return compilers.CompileReport{}, fmt.Errorf("unable to copy binary: %w", err)
			}
		}
	}
	return compilers.CompileReport{
		ExitCode:   report.ExitCode,
		UsedTime:   report.Time,
		UsedMemory: report.Memory,
		Log:        log.String(),
	}, nil
}

func (c *compiler) CreateExecutable(ctx context.Context, binaryPath string) (compilers.Executable, error) {
	if c.config.Execute == nil {
		return nil, fmt.Errorf("compiler has empty execute config")
	}
	exe := executable{
		compiler: c,
		config:   *c.config.Execute,
	}
	if c.config.Execute.Binary == nil {
		return &exe, nil
	}
	layerPath, err := os.MkdirTemp("", "layer-*")
	if err != nil {
		return nil, err
	}
	layerBinaryPath := filepath.Join(
		layerPath, c.config.Execute.Workdir, *c.config.Execute.Binary,
	)
	if err := copyFileRec(binaryPath, layerBinaryPath); err != nil {
		return nil, fmt.Errorf("unable to copy binary: %w", err)
	}
	exe.layer = layerPath
	return &exe, nil
}

type compilerManager struct {
	files     *managers.FileManager
	cacheDir  string
	safeexec  *safeexec.Manager
	compilers *models.CompilerStore
	settings  *models.SettingStore
	images    map[int64]futures.Future[string]
	logger    *logs.Logger
	mutex     sync.Mutex
}

func newCompilerManager(
	files *managers.FileManager,
	cacheDir string,
	safeexec *safeexec.Manager,
	core *core.Core,
) (*compilerManager, error) {
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return nil, err
	}
	return &compilerManager{
		files:     files,
		cacheDir:  cacheDir,
		safeexec:  safeexec,
		compilers: core.Compilers,
		settings:  core.Settings,
		images:    map[int64]futures.Future[string]{},
		logger:    core.Logger(),
	}, nil
}

func (m *compilerManager) GetCompilerName(name string) (string, error) {
	setting, err := m.settings.GetByKey("invoker.compilers." + name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("cannot get compiler %q", name)
		}
		return "", err
	}
	return setting.Value, nil
}

func (m *compilerManager) GetCompiler(ctx context.Context, name string) (compilers.Compiler, error) {
	compiler, err := m.compilers.GetByName(name)
	if err != nil {
		return nil, err
	}
	return m.DownloadCompiler(ctx, compiler)
}

func (m *compilerManager) Logger() *logs.Logger {
	return m.logger
}

func (m *compilerManager) Release() {}

func (m *compilerManager) DownloadCompiler(ctx context.Context, c models.Compiler) (compilers.Compiler, error) {
	config, err := c.GetConfig()
	if err != nil {
		return nil, err
	}
	imagePath, err := m.downloadImageAsync(ctx, c.ImageID).Get(ctx)
	if err != nil {
		return nil, err
	}
	return &compiler{
		safeexec: m.safeexec,
		layer:    imagePath,
		name:     c.Name,
		config:   config,
	}, nil
}

func (m *compilerManager) downloadImageAsync(ctx context.Context, imageID int64) futures.Future[string] {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if image, ok := m.images[imageID]; ok {
		return image
	}
	future, setResult := futures.New[string]()
	m.images[imageID] = future
	go func() {
		image, err := m.runDownloadImage(ctx, imageID)
		if err != nil {
			m.deleteImage(imageID)
		}
		setResult(image, err)
	}()
	return future
}

func (m *compilerManager) runDownloadImage(ctx context.Context, imageID int64) (string, error) {
	imageFile, err := m.files.DownloadFile(ctx, imageID)
	if err != nil {
		return "", err
	}
	defer func() { _ = imageFile.Close() }()
	localImagePath := filepath.Join(m.cacheDir, fmt.Sprintf("image-%d.tar.gz", imageID))
	_ = os.Remove(localImagePath)
	imagePath := filepath.Join(m.cacheDir, fmt.Sprintf("image-%d", imageID))
	_ = os.RemoveAll(imagePath)
	if file, ok := imageFile.(*os.File); ok {
		localImagePath = file.Name()
	} else {
		localImageFile, err := os.Create(localImagePath)
		if err != nil {
			return "", err
		}
		defer func() {
			_ = localImageFile.Close()
			_ = os.Remove(localImagePath)
		}()
		if _, err := io.Copy(localImageFile, imageFile); err != nil {
			return "", err
		}
		if err := localImageFile.Close(); err != nil {
			return "", err
		}
	}
	if err := archives.ExtractTarGz(localImagePath, imagePath); err != nil {
		return "", fmt.Errorf("cannot extract image: %w", err)
	}
	return imagePath, nil
}

func (m *compilerManager) deleteImage(imageID int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	imagePath := filepath.Join(m.cacheDir, fmt.Sprintf("image-%d", imageID))
	_ = os.RemoveAll(imagePath)
	delete(m.images, imageID)
}
