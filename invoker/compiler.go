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
	"github.com/udovin/solve/pkg"
	"github.com/udovin/solve/pkg/logs"
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

type ExecuteOptions struct {
	Binary      string
	Args        []string
	InputFiles  []MountFile
	OutputFiles []MountFile
	TimeLimit   time.Duration
	MemoryLimit int64
}

type Compiler interface {
	Name() string
	Compile(ctx context.Context, options CompileOptions) (CompileReport, error)
	Execute(ctx context.Context, options ExecuteOptions) (ExecuteReport, error)
}

type compiler struct {
	safeexec *safeexecProcessor
	name     string
	config   models.CompilerConfig
	path     string
}

func (c *compiler) Name() string {
	return c.name
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
	log := strings.Builder{}
	config := safeexecProcessConfig{
		Layers:      []string{c.path},
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
			process.GetUpperDir(),
			c.config.Compile.Workdir,
			*c.config.Compile.Source,
		)
		if err := copyFileRec(options.Source, path); err != nil {
			return CompileReport{}, fmt.Errorf("unable to write source: %w", err)
		}
	}
	for _, file := range options.InputFiles {
		path := filepath.Join(
			process.GetUpperDir(),
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
				process.GetUpperDir(),
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

const (
	stdinFile  = "stdin"
	stdoutFile = "stdout"
	stderrFile = "stderr"
)

func (c *compiler) Execute(ctx context.Context, options ExecuteOptions) (ExecuteReport, error) {
	if c.config.Execute == nil {
		return ExecuteReport{}, nil
	}
	var stdin io.Reader
	for _, input := range options.InputFiles {
		if input.Target != stdinFile {
			continue
		}
		file, err := os.Open(input.Source)
		if err != nil {
			return ExecuteReport{}, fmt.Errorf("cannot open input file: %w", err)
		}
		defer func() { _ = file.Close() }()
		stdin = file
		break
	}
	var stdout io.Writer
	var stderr io.Writer
	for _, output := range options.OutputFiles {
		if output.Target != stdoutFile && output.Target != stderrFile {
			continue
		}
		file, err := os.Create(output.Source)
		if err != nil {
			return ExecuteReport{}, fmt.Errorf("cannot create output file: %w", err)
		}
		defer func() { _ = file.Close() }()
		if output.Target == stdoutFile {
			stdout = file
		} else {
			stderr = file
		}
		break
	}
	executeArgs := append(strings.Fields(c.config.Execute.Command), options.Args...)
	config := safeexecProcessConfig{
		Layers:      []string{c.path},
		Command:     executeArgs,
		Environ:     c.config.Execute.Environ,
		Workdir:     c.config.Execute.Workdir,
		Stdin:       stdin,
		Stdout:      stdout,
		Stderr:      stderr,
		TimeLimit:   options.TimeLimit,
		MemoryLimit: options.MemoryLimit,
	}
	process, err := c.safeexec.Create(ctx, config)
	if err != nil {
		return ExecuteReport{}, fmt.Errorf("unable to create compiler: %w", err)
	}
	if c.config.Execute.Binary != nil {
		path := filepath.Join(
			process.GetUpperDir(),
			c.config.Execute.Workdir,
			*c.config.Execute.Binary,
		)
		if err := copyFileRec(options.Binary, path); err != nil {
			return ExecuteReport{}, fmt.Errorf("unable to write binary: %w", err)
		}
	}
	for _, file := range options.InputFiles {
		if file.Target == stdinFile {
			continue
		}
		path := filepath.Join(
			process.GetUpperDir(),
			c.config.Execute.Workdir,
			file.Target,
		)
		if err := copyFileRec(file.Source, path); err != nil {
			return ExecuteReport{}, fmt.Errorf("unable to write file: %w", err)
		}
	}
	defer func() { _ = process.Release() }()
	if err := process.Start(); err != nil {
		return ExecuteReport{}, fmt.Errorf("cannot start compiler: %w", err)
	}
	report, err := process.Wait()
	if err != nil {
		return ExecuteReport{}, err
	}
	if report.ExitCode == 0 {
		for _, output := range options.OutputFiles {
			if output.Target == stdoutFile || output.Target == stderrFile {
				continue
			}
			containerPath := filepath.Join(
				process.GetUpperDir(),
				c.config.Execute.Workdir,
				output.Target,
			)
			if err := copyFileRec(containerPath, output.Source); err != nil {
				return ExecuteReport{}, fmt.Errorf("unable to copy binary: %w", err)
			}
		}
	}
	return ExecuteReport{
		ExitCode:   report.ExitCode,
		UsedTime:   report.Time,
		UsedMemory: report.Memory,
	}, nil
}

type compilerManager struct {
	files     *managers.FileManager
	cacheDir  string
	safeexec  *safeexecProcessor
	compilers *models.CompilerStore
	settings  *models.SettingStore
	images    map[int64]futures.Future[string]
	logger    *logs.Logger
	mutex     sync.Mutex
}

func newCompilerManager(
	files *managers.FileManager,
	cacheDir string,
	safeexec *safeexecProcessor,
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
		path:     imagePath,
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
	if err := pkg.ExtractTarGz(localImagePath, imagePath); err != nil {
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
