package cache

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/compilers"
	"github.com/udovin/solve/internal/pkg/safeexec"
	"github.com/udovin/solve/internal/pkg/utils"
)

const (
	maxDiagnosticsFileSize = 8 * 1024
	maxCompileLogSize      = 8 * 1024
	maxDiagnosticsCount    = 100
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
	log := utils.NewTruncateBuffer(maxCompileLogSize)
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
	compileReport := compilers.CompileReport{
		ExitCode:   report.ExitCode,
		UsedTime:   report.Time,
		UsedMemory: report.Memory,
		Log:        log.String(),
	}
	if c.config.Compile.Diagnostics != nil {
		diagnosticsPath := filepath.Join(
			process.UpperDir(),
			c.config.Compile.Workdir,
			*c.config.Compile.Diagnostics,
		)
		compileReport.Diagnostics = readDiagnostics(diagnosticsPath)
	}
	return compileReport, nil
}

// readDiagnostics reads NDJSON file (one diagnostic per line).
// Stops at size limit, count limit, or first invalid line.
func readDiagnostics(path string) []models.Diagnostic {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(io.LimitReader(file, maxDiagnosticsFileSize))
	scanner.Buffer(make([]byte, 0, 4096), maxDiagnosticsFileSize)
	var diagnostics []models.Diagnostic
	for scanner.Scan() {
		if len(diagnostics) >= maxDiagnosticsCount {
			break
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var d models.Diagnostic
		if err := json.Unmarshal(line, &d); err != nil {
			// Skip invalid lines — could be a truncated last entry.
			break
		}
		diagnostics = append(diagnostics, d)
	}
	return diagnostics
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
