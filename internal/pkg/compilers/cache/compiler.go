package cache

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/compilers"
	"github.com/udovin/solve/internal/pkg/safeexec"
)

type compiler struct {
	name     string
	layer    string
	config   models.CompilerConfig
	safeexec *safeexec.Manager
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

func copyFileRec(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
		return err
	}
	return copyFile(source, target)
}

func copyFile(source, target string) error {
	r, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	stat, err := r.Stat()
	if err != nil {
		return err
	}
	w, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return os.Chmod(w.Name(), stat.Mode())
}

type truncateBuffer struct {
	buffer strings.Builder
	limit  int
	mutex  sync.Mutex
}

func (b *truncateBuffer) String() string {
	return fixUTF8String(b.buffer.String())
}

func (b *truncateBuffer) Write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	l := len(p)
	if b.buffer.Len()+l > b.limit {
		p = p[:b.limit-b.buffer.Len()]
	}
	if len(p) == 0 {
		return l, nil
	}
	n, err := b.buffer.Write(p)
	if err != nil {
		return n, err
	}
	return l, nil
}

func fixUTF8String(s string) string {
	return strings.ReplaceAll(strings.ToValidUTF8(s, ""), "\u0000", "")
}
