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
	"time"

	"github.com/udovin/algo/futures"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg/archives"
	"github.com/udovin/solve/pkg/logs"
	"github.com/udovin/solve/pkg/safeexec"
)

type MountFile struct {
	Source string
	Target string
}

type CompileReport struct {
	ExitCode   int
	UsedTime   time.Duration
	UsedMemory int64
	Log        string
}

func (r CompileReport) Success() bool {
	return r.ExitCode == 0
}

type CompileOptions struct {
	Source      string
	Target      string
	InputFiles  []MountFile
	TimeLimit   time.Duration
	MemoryLimit int64
}

type ExecuteReport struct {
	ExitCode   int
	UsedTime   time.Duration
	UsedMemory int64
}

func (r ExecuteReport) Success() bool {
	return r.ExitCode == 0
}

type CompilerProcess interface {
	Start() error
	Wait() (ExecuteReport, error)
	Release() error
}

type Compiler interface {
	Name() string
	Compile(ctx context.Context, options CompileOptions) (CompileReport, error)
	CreateExecutable(binaryPath string) (Executable, error)
}

type CompilerManager interface {
	GetCompiler(ctx context.Context, name string) (Compiler, error)
	GetCompilerName(name string) (string, error)
	Logger() *logs.Logger
}

type compiler struct {
	safeexec *safeexec.Manager
	name     string
	config   models.CompilerConfig
	layer    string
}

func (c *compiler) Name() string {
	return c.name
}

type truncateBuffer struct {
	strings.Builder
	limit int
	mutex sync.Mutex
}

func (b *truncateBuffer) Write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	l := len(p)
	if b.Builder.Len()+l > b.limit {
		p = p[:b.limit-b.Builder.Len()]
	}
	if len(p) == 0 {
		return l, nil
	}
	n, err := b.Builder.Write(p)
	if err != nil {
		return n, err
	}
	return l, nil
}

func (c *compiler) Compile(
	ctx context.Context, options CompileOptions,
) (CompileReport, error) {
	if c.config.Compile == nil {
		if err := copyFileRec(options.Source, options.Target); err != nil {
			return CompileReport{}, fmt.Errorf("unable to copy source: %w", err)
		}
		return CompileReport{}, nil
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
		return CompileReport{}, fmt.Errorf("unable to create compiler: %w", err)
	}
	if c.config.Compile.Source != nil {
		path := filepath.Join(
			process.UpperDir(),
			c.config.Compile.Workdir,
			*c.config.Compile.Source,
		)
		if err := copyFileRec(options.Source, path); err != nil {
			return CompileReport{}, fmt.Errorf("unable to write source: %w", err)
		}
	}
	for _, file := range options.InputFiles {
		path := filepath.Join(
			process.UpperDir(),
			c.config.Compile.Workdir,
			file.Target,
		)
		if err := copyFileRec(file.Source, path); err != nil {
			return CompileReport{}, fmt.Errorf("unable to write file: %w", err)
		}
	}
	defer func() { _ = process.Release() }()
	if err := process.Start(); err != nil {
		return CompileReport{}, fmt.Errorf("cannot start compiler: %w", err)
	}
	report, err := process.Wait()
	if err != nil {
		return CompileReport{}, err
	}
	if report.ExitCode == 0 {
		if c.config.Compile.Binary != nil {
			containerBinaryPath := filepath.Join(
				process.UpperDir(),
				c.config.Compile.Workdir,
				*c.config.Compile.Binary,
			)
			if err := copyFileRec(containerBinaryPath, options.Target); err != nil {
				return CompileReport{}, fmt.Errorf("unable to copy binary: %w", err)
			}
		}
	}
	return CompileReport{
		ExitCode:   report.ExitCode,
		UsedTime:   report.Time,
		UsedMemory: report.Memory,
		Log:        log.String(),
	}, nil
}

func (c *compiler) CreateExecutable(binaryPath string) (Executable, error) {
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

func (m *compilerManager) GetCompiler(ctx context.Context, name string) (Compiler, error) {
	compiler, err := m.compilers.GetByName(name)
	if err != nil {
		return nil, err
	}
	return m.DownloadCompiler(ctx, compiler)
}

func (m *compilerManager) Logger() *logs.Logger {
	return m.logger
}

func (m *compilerManager) DownloadCompiler(ctx context.Context, c models.Compiler) (Compiler, error) {
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
