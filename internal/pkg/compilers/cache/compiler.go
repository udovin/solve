package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/compilers"
	"github.com/udovin/solve/internal/pkg/safeexec"
	"github.com/udovin/solve/internal/pkg/utils"
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
		if err := utils.CopyFileRec(options.Target, options.Source); err != nil {
			return compilers.CompileReport{}, fmt.Errorf("unable to copy source: %w", err)
		}
		return compilers.CompileReport{}, nil
	}
	log := utils.NewTruncateBuffer(2048)
	config := safeexec.ProcessConfig{
		Layers:      []string{c.layer},
		Command:     strings.Fields(c.config.Compile.Command),
		Environ:     c.config.Compile.Environ,
		Workdir:     c.config.Compile.Workdir,
		Stdout:      log,
		Stderr:      log,
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
		if err := utils.CopyFileRec(path, options.Source); err != nil {
			return compilers.CompileReport{}, fmt.Errorf("unable to write source: %w", err)
		}
	}
	for _, file := range options.InputFiles {
		path := filepath.Join(
			process.UpperDir(),
			c.config.Compile.Workdir,
			file.Target,
		)
		if err := utils.CopyFileRec(path, file.Source); err != nil {
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
			if err := utils.CopyFileRec(options.Target, containerBinaryPath); err != nil {
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
	if err := utils.CopyFileRec(layerBinaryPath, binaryPath); err != nil {
		return nil, fmt.Errorf("unable to copy binary: %w", err)
	}
	exe.layer = layerPath
	return &exe, nil
}
