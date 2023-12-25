package compilers

import (
	"context"
	"io"
	"time"

	"github.com/udovin/solve/internal/pkg/safeexec"
)

type ExecuteOptions struct {
	Args        []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	TimeLimit   time.Duration
	MemoryLimit int64
}

type Executable interface {
	CreateProcess(ctx context.Context, options ExecuteOptions) (*safeexec.Process, error)
	Release() error
}

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

type Compiler interface {
	Name() string
	Compile(ctx context.Context, options CompileOptions) (CompileReport, error)
	CreateExecutable(ctx context.Context, binaryPath string) (Executable, error)
}
