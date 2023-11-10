package safeexec

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Process struct {
	name       string
	path       string
	cgroupPath string
	workdir    string
	cmd        *exec.Cmd
}

type Report struct {
	Time     time.Duration
	RealTime time.Duration
	Memory   int64
	ExitCode int
}

func (p *Process) Start() error {
	return p.cmd.Start()
}

func (p *Process) UpperDir() string {
	return filepath.Join(p.path, "upper")
}

func (p *Process) UpperPath(path string) string {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(p.workdir, path)
	}
	if !filepath.IsAbs(path) {
		// This should never have happened.
		panic(fmt.Errorf("path %q is not absolute", path))
	}
	return filepath.Join(p.UpperDir(), path)
}

func (p *Process) Wait() (Report, error) {
	if err := p.cmd.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		if v, ok := err.(*exec.ExitError); ok && v.ProcessState != nil {
			if status, ok := v.ProcessState.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() && status.Signal() == syscall.SIGTERM {
					return Report{ExitCode: -1}, nil
				}
			}
		}
		return Report{}, err
	}
	file, err := os.Open(filepath.Join(p.path, "report.txt"))
	if err != nil {
		return Report{}, err
	}
	report := Report{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			return Report{}, fmt.Errorf("cannot read report")
		}
		switch parts[0] {
		case "exit_code":
			value, err := strconv.ParseInt(parts[1], 10, 32)
			if err != nil {
				return Report{}, fmt.Errorf("cannot parse exit_code: %w", err)
			}
			report.ExitCode = int(value)
		case "time":
			value, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return Report{}, fmt.Errorf("cannot parse time: %w", err)
			}
			report.Time = time.Duration(value) * time.Millisecond
		case "real_time":
			value, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return Report{}, fmt.Errorf("cannot parse time: %w", err)
			}
			report.RealTime = time.Duration(value) * time.Millisecond
		case "memory":
			value, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return Report{}, fmt.Errorf("cannot parse memory: %w", err)
			}
			report.Memory = value
		}
	}
	return report, nil
}

// Release releases all associatet resources with process.
func (p *Process) Release() error {
	if p.cmd != nil {
		if p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
		}
		_ = p.cmd.Wait()
	}
	_ = syscall.Rmdir(p.cgroupPath)
	return os.RemoveAll(p.path)
}
