package invoker

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg/safeexec"
)

type ExecuteOptions struct {
	Args        []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	TimeLimit   time.Duration
	MemoryLimit int64
	InputFiles  []MountFile
	OutputFiles []MountFile
	// deprecated.
	Binary string
}

type Executable interface {
	CreateProcess(ctx context.Context, options ExecuteOptions) (*safeexec.Process, error)
	Release() error
}

type executable struct {
	safexec      *safeexec.Manager
	layer        string
	parentLayers []string
	config       models.CompilerCommandConfig
}

func (e *executable) Release() error {
	if e.layer != "" {
		return os.RemoveAll(e.layer)
	}
	return nil
}

func (e *executable) CreateProcess(
	ctx context.Context, options ExecuteOptions,
) (*safeexec.Process, error) {
	config := safeexec.ProcessConfig{
		Layers:      append([]string{e.layer}, e.parentLayers...),
		Stdin:       options.Stdin,
		Stdout:      options.Stdout,
		Stderr:      options.Stderr,
		Environ:     e.config.Environ,
		Workdir:     e.config.Workdir,
		Command:     append(strings.Fields(e.config.Command), options.Args...),
		TimeLimit:   options.TimeLimit,
		MemoryLimit: options.MemoryLimit,
	}
	process, err := e.safexec.Create(ctx, config)
	if err != nil {
		return nil, err
	}
	return process, nil
}
