package invoker

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
	UsedTime   int64
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
	TimeLimit   int64
	MemoryLimit int64
}

type ExecuteReport struct {
	ExitCode   int
	UsedTime   int64
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
	TimeLimit   int64
	MemoryLimit int64
}

type Compiler interface {
	Name() string
	Compile(ctx context.Context, options CompileOptions) (CompileReport, error)
	Execute(ctx context.Context, options ExecuteOptions) (ExecuteReport, error)
}

type compiler struct {
	factory *factory
	name    string
	config  models.CompilerConfig
	path    string
}

func (c *compiler) Name() string {
	return c.name
}

type waitResult struct {
	State *os.ProcessState
	Err   error
}

type executeResult struct {
	MaxMemory int64
	Duration  time.Duration
	ExitCode  int
}

func execute(container *container, memoryLimit int64, timeLimit time.Duration) (executeResult, error) {
	process, err := container.Start()
	if err != nil {
		return executeResult{}, fmt.Errorf("unable to start compiler: %w", err)
	}
	defer func() { _ = process.Signal(os.Kill) }()
	start := time.Now()
	waitChan := make(chan waitResult, 1)
	defer close(waitChan)
	var waiter sync.WaitGroup
	defer waiter.Wait()
	waiter.Add(1)
	go func() {
		defer waiter.Done()
		state, err := process.Wait()
		waitChan <- waitResult{State: state, Err: err}
	}()
	deadlineCtx, cancel := context.WithDeadline(context.Background(), start.Add(timeLimit+2*time.Millisecond))
	defer cancel()
	var maxMemory int64
	waiter.Add(1)
	go func() {
		defer waiter.Done()
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		stats, err := container.container.Stats()
		if err == nil && stats.CgroupStats == nil {
			memory := stats.CgroupStats.MemoryStats.SwapUsage.Usage
			if int64(memory) > maxMemory {
				atomic.StoreInt64(&maxMemory, int64(memory))
			}
		}
		for {
			select {
			case <-deadlineCtx.Done():
				return
			case <-ticker.C:
				stats, err := container.container.Stats()
				if err != nil {
					continue
				}
				if stats.CgroupStats == nil {
					continue
				}
				memory := stats.CgroupStats.MemoryStats.SwapUsage.Usage
				if int64(memory) > maxMemory {
					atomic.StoreInt64(&maxMemory, int64(memory))
				}
			}
		}
	}()
	notifyOOM, err := container.container.NotifyOOM()
	if err != nil {
		return executeResult{}, fmt.Errorf("cannot setup oom notify: %w", err)
	}
	select {
	case result := <-waitChan:
		duration := time.Since(start)
		memory := atomic.LoadInt64(&maxMemory)
		state, err := result.State, result.Err
		if err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				return executeResult{}, err
			}
		}
		exitCode := state.ExitCode()
		if !state.Exited() {
			exitCode = -1
		}
		return executeResult{
			Duration:  duration,
			MaxMemory: memory,
			ExitCode:  exitCode,
		}, nil

	case <-notifyOOM:
		duration := time.Since(start)
		memory := atomic.LoadInt64(&maxMemory)
		if memory < memoryLimit {
			memory = memoryLimit + 1024
		}
		return executeResult{
			Duration:  duration,
			MaxMemory: memory,
			ExitCode:  -1,
		}, nil
	case <-deadlineCtx.Done():
		duration := time.Since(start)
		memory := atomic.LoadInt64(&maxMemory)
		return executeResult{
			Duration:  duration,
			MaxMemory: memory,
			ExitCode:  -1,
		}, nil
	}
}

func (c *compiler) Compile(
	ctx context.Context, options CompileOptions,
) (CompileReport, error) {
	stderr := strings.Builder{}
	containerConfig := containerConfig{
		Layers: []string{c.path},
		Init: processConfig{
			Args:   strings.Fields(c.config.Compile.Command),
			Env:    c.config.Compile.Environ,
			Dir:    c.config.Compile.Workdir,
			Stderr: &stderr,
		},
		MemoryLimit: options.MemoryLimit + 1024,
	}
	container, err := c.factory.Create(containerConfig)
	if err != nil {
		return CompileReport{}, fmt.Errorf("unable to create compiler: %w", err)
	}
	if c.config.Compile.Source != nil {
		path := filepath.Join(
			container.GetUpperDir(),
			c.config.Compile.Workdir,
			*c.config.Compile.Source,
		)
		if err := copyFileRec(options.Source, path); err != nil {
			return CompileReport{}, fmt.Errorf("unable to write source: %w", err)
		}
	}
	for _, file := range options.InputFiles {
		path := filepath.Join(
			container.GetUpperDir(),
			c.config.Compile.Workdir,
			file.Target,
		)
		if err := copyFileRec(file.Source, path); err != nil {
			return CompileReport{}, fmt.Errorf("unable to write file: %w", err)
		}
	}
	defer func() { _ = container.Destroy() }()
	timeLimit := time.Duration(options.TimeLimit) * time.Millisecond
	result, err := execute(container, options.MemoryLimit, timeLimit)
	if err != nil {
		return CompileReport{}, err
	}
	if result.ExitCode == 0 {
		if c.config.Compile.Binary != nil {
			containerBinaryPath := filepath.Join(
				container.GetUpperDir(),
				c.config.Compile.Workdir,
				*c.config.Compile.Binary,
			)
			if err := copyFileRec(containerBinaryPath, options.Target); err != nil {
				return CompileReport{}, fmt.Errorf("unable to copy binary: %w", err)
			}
		}
	}
	return CompileReport{
		ExitCode:   result.ExitCode,
		UsedTime:   result.Duration.Milliseconds(),
		UsedMemory: result.MaxMemory,
		Log:        stderr.String(),
	}, nil
}

const (
	stdinFile  = "stdin"
	stdoutFile = "stdout"
	stderrFile = "stderr"
)

func (c *compiler) Execute(ctx context.Context, options ExecuteOptions) (ExecuteReport, error) {
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
	containerConfig := containerConfig{
		Layers: []string{c.path},
		Init: processConfig{
			Args:   executeArgs,
			Env:    c.config.Execute.Environ,
			Dir:    c.config.Execute.Workdir,
			Stdin:  stdin,
			Stdout: stdout,
			Stderr: stderr,
		},
		MemoryLimit: options.MemoryLimit + 1024,
	}
	container, err := c.factory.Create(containerConfig)
	if err != nil {
		return ExecuteReport{}, fmt.Errorf("unable to create compiler: %w", err)
	}
	if c.config.Execute.Binary != nil {
		path := filepath.Join(
			container.GetUpperDir(),
			c.config.Execute.Workdir,
			*c.config.Execute.Binary,
		)
		if err := copyFileRec(options.Binary, path); err != nil {
			return ExecuteReport{}, fmt.Errorf("unable to write source: %w", err)
		}
	}
	for _, file := range options.InputFiles {
		if file.Target == stdinFile {
			continue
		}
		path := filepath.Join(
			container.GetUpperDir(),
			c.config.Execute.Workdir,
			file.Target,
		)
		if err := copyFileRec(file.Source, path); err != nil {
			return ExecuteReport{}, fmt.Errorf("unable to write file: %w", err)
		}
	}
	defer func() { _ = container.Destroy() }()
	timeLimit := time.Duration(options.TimeLimit) * time.Millisecond
	result, err := execute(container, options.MemoryLimit, timeLimit)
	if err != nil {
		return ExecuteReport{}, err
	}
	if result.ExitCode == 0 {
		for _, output := range options.OutputFiles {
			if output.Target == stdoutFile || output.Target == stderrFile {
				continue
			}
			containerPath := filepath.Join(
				container.GetUpperDir(),
				c.config.Execute.Workdir,
				output.Target,
			)
			if err := copyFileRec(containerPath, output.Source); err != nil {
				return ExecuteReport{}, fmt.Errorf("unable to copy binary: %w", err)
			}
		}
	}
	return ExecuteReport{
		ExitCode:   result.ExitCode,
		UsedTime:   result.Duration.Milliseconds(),
		UsedMemory: result.MaxMemory,
	}, nil
}

type compilerManager struct {
	files     *managers.FileManager
	cacheDir  string
	factory   *factory
	compilers *models.CompilerStore
	settings  *models.SettingStore
	images    map[int64]futures.Future[string]
	logger    *logs.Logger
	mutex     sync.Mutex
}

func newCompilerManager(
	files *managers.FileManager,
	cacheDir string,
	factory *factory,
	core *core.Core,
) (*compilerManager, error) {
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return nil, err
	}
	return &compilerManager{
		files:     files,
		cacheDir:  cacheDir,
		factory:   factory,
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
		factory: m.factory,
		path:    imagePath,
		name:    c.Name,
		config:  config,
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
