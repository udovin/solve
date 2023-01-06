package invoker

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/udovin/algo/futures"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg"
)

type CompileReport struct {
	Success bool
	Log     string
}

type Compiler interface {
	Compile(ctx context.Context, source, target string, additionalFiles ...string) (CompileReport, error)
}

type compiler struct {
	factory *factory
	config  models.CompilerConfig
	path    string
}

func (c *compiler) Compile(
	ctx context.Context, source, target string, additionalFiles ...string,
) (CompileReport, error) {
	stdout := strings.Builder{}
	containerConfig := containerConfig{
		Layers: []string{c.path},
		Init: processConfig{
			Args:   strings.Fields(c.config.Compile.Command),
			Env:    c.config.Compile.Environ,
			Dir:    c.config.Compile.Workdir,
			Stdout: &stdout,
		},
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
		if err := copyFileRec(source, path); err != nil {
			return CompileReport{}, fmt.Errorf("unable to write solution: %w", err)
		}
	}
	for _, file := range additionalFiles {
		path := filepath.Join(
			container.GetUpperDir(),
			c.config.Compile.Workdir,
			filepath.Base(file),
		)
		if err := copyFileRec(file, path); err != nil {
			return CompileReport{}, fmt.Errorf("unable to write additional file: %w", err)
		}
	}
	defer func() { _ = container.Destroy() }()
	process, err := container.Start()
	if err != nil {
		return CompileReport{}, fmt.Errorf("unable to start compiler: %w", err)
	}
	state, err := process.Wait()
	if err != nil {
		if err, ok := err.(*exec.ExitError); !ok {
			return CompileReport{}, fmt.Errorf("unable to wait compiler: %w", err)
		} else {
			return CompileReport{
				Success: false,
				Log:     stdout.String(),
			}, nil
		}
	}
	if !state.Exited() || state.ExitCode() != 0 {
		return CompileReport{
			Success: false,
			Log:     stdout.String(),
		}, nil
	}
	if c.config.Compile.Binary != nil {
		containerBinaryPath := filepath.Join(
			container.GetUpperDir(),
			c.config.Compile.Workdir,
			*c.config.Compile.Binary,
		)
		if err := copyFileRec(containerBinaryPath, target); err != nil {
			return CompileReport{}, fmt.Errorf("unable to copy binary: %w", err)
		}
	}
	report := CompileReport{
		Success: true,
		Log:     stdout.String(),
	}
	return report, nil
}

type compilerManager struct {
	files     *managers.FileManager
	cacheDir  string
	factory   *factory
	compilers *models.CompilerStore
	settings  *models.SettingStore
	images    map[int64]futures.Future[string]
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
	}, nil
}

func (m *compilerManager) GetCompiler(ctx context.Context, name string) (Compiler, error) {
	setting, err := m.settings.GetByKey("invoker.compilers." + name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cannot get compiler %q", name)
		}
		return nil, err
	}
	id, err := strconv.ParseInt(setting.Value, 10, 64)
	if err != nil {
		return nil, err
	}
	compiler, err := m.compilers.Get(id)
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
	return &compiler{factory: m.factory, path: imagePath, config: config}, nil
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
