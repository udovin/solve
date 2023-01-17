package invoker

import (
	"time"
	"fmt"
	"context"
	"os/exec"
	"os"
	"path/filepath"
	"crypto/rand"
	"syscall"
	"encoding/hex"
)

type executeFileConfig struct {
	Source string
	Target string
}

type safeexecProcessConfig struct {
	TimeLimit   time.Duration
	MemoryLimit int64
	StdinPath   string
	StdoutPath  string
	StderrPath  string
	ImagePath   string
	Workdir     string
	InputFiles  []executeFileConfig
	OutputFiles []executeFileConfig
}

type safeexecProcess struct {
	name       string
	path       string
	cgroupPath string
	cmd        *exec.Cmd
}

func (p *safeexecProcess) Release() error {
	if p.cmd != nil {
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
	}
	_ = syscall.Rmdir(p.cgroupPath)
	return os.RemoveAll(p.path)
}

type safeexec struct {
	path             string
	cgroupParentPath string
	executionPath    string
}

func (m *safeexec) Execute(ctx context.Context, config safeexecProcessConfig) (*safeexecProcess, error) {
	process, err := m.prepareProcess()
	if err != nil {
		return nil, err
	}
	process.cgroupPath = filepath.Join(m.cgroupParentPath, process.name)
	upperdir := filepath.Join(process.path, "upper")
	if err := os.Mkdir(upperdir, 0777); err != nil {
		_ = process.Release()
		return nil, err
	}
	var args []string
	args = append(args, "--time-limit", fmt.Sprint(config.TimeLimit.Milliseconds()))
	args = append(args, "--memory-limit", fmt.Sprint(config.MemoryLimit))
	args = append(args, "--lowerdir", config.ImagePath)
	args = append(args, "--upperdir", upperdir)
	args = append(args, "--cgroup-parent", m.cgroupParentPath)
	args = append(args, "--cgroup-name", process.name)
	if len(config.Workdir) > 0 {
		args = append(args, "--workdir", config.Workdir)
	}
	if len(config.StdinPath) > 0 {
		args = append(args, "--stdin", config.StdinPath)
	}
	if len(config.StdoutPath) > 0 {
		args = append(args, "--stdout", config.StdoutPath)
	}
	if len(config.StderrPath) > 0 {
		args = append(args, "--stderr", config.StderrPath)
	}
	cmd := exec.CommandContext(ctx, m.path, args...)
	process.cmd = cmd
	return process, nil
}

func (m *safeexec) prepareProcess() (*safeexecProcess, error) {
	for i := 0; i < 100; i++ {
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			return nil, err
		}
		name := hex.EncodeToString(bytes)
		path := filepath.Join(m.executionPath, name)
		if err := os.MkdirAll(path, 0777); err != nil {
			if os.IsExist(err) {
				continue
			}
			return nil, err
		}
		return &safeexecProcess{name: name, path: path}, nil
	}
	return nil, fmt.Errorf("cannot prepare process")
}

func newSafeexec() (*safeexec, error) {
	return &safeexec{}, nil
}
