package invoker

import (
	"context"
	"os"
	"strings"

	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/compilers"
	"github.com/udovin/solve/internal/pkg/safeexec"
)

type executable struct {
	compiler *compiler
	layer    string
	config   models.CompilerCommandConfig
}

func (e *executable) CreateProcess(
	ctx context.Context, options compilers.ExecuteOptions,
) (*safeexec.Process, error) {
	config := safeexec.ProcessConfig{
		Layers:      e.getLayers(),
		Stdin:       options.Stdin,
		Stdout:      options.Stdout,
		Stderr:      options.Stderr,
		Environ:     e.config.Environ,
		Workdir:     e.config.Workdir,
		Command:     append(strings.Fields(e.config.Command), options.Args...),
		TimeLimit:   options.TimeLimit,
		MemoryLimit: options.MemoryLimit,
	}
	process, err := e.compiler.safeexec.Create(ctx, config)
	if err != nil {
		return nil, err
	}
	return process, nil
}

func (e *executable) Release() error {
	if e.layer == "" {
		return nil
	}
	return os.RemoveAll(e.layer)
}

func (e *executable) getLayers() []string {
	if e.layer == "" {
		return []string{e.compiler.layer}
	}
	return []string{e.layer, e.compiler.layer}
}
