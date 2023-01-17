package invoker

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
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
	Command     []string
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
		if p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
		}
		_ = p.cmd.Wait()
	}
	_ = syscall.Rmdir(p.cgroupPath)
	return os.RemoveAll(p.path)
}

type safeexecProcessor struct {
	path          string
	executionPath string
	cgroupPath    string
}

func (m *safeexecProcessor) Execute(ctx context.Context, config safeexecProcessConfig) (*safeexecProcess, error) {
	process, err := m.prepareProcess()
	if err != nil {
		return nil, err
	}
	var args []string
	args = append(args, "--time-limit", fmt.Sprint(config.TimeLimit.Milliseconds()))
	args = append(args, "--memory-limit", fmt.Sprint(config.MemoryLimit))
	args = append(args, "--overlay-lowerdir", config.ImagePath)
	args = append(args, "--overlay-upperdir", filepath.Join(process.path, "upper"))
	args = append(args, "--overlay-workdir", filepath.Join(process.path, "workdir"))
	args = append(args, "--rootfs", filepath.Join(process.path, "rootfs"))
	args = append(args, "--cgroup-path", process.cgroupPath)
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
	args = append(args, config.Command...)
	cmd := exec.CommandContext(ctx, m.path, args...)
	process.cmd = cmd
	return process, nil
}

func (m *safeexecProcessor) createProcessName() (string, error) {
	for i := 0; i < 100; i++ {
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}
		name := hex.EncodeToString(bytes)
		path := filepath.Join(m.executionPath, name)
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", err
		}
		return name, nil
	}
	return "", fmt.Errorf("cannot prepare process")
}

func (m *safeexecProcessor) prepareProcess() (*safeexecProcess, error) {
	name, err := m.createProcessName()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(m.executionPath, name)
	cgroupPath := filepath.Join(m.cgroupPath, name)
	if err := syscall.Rmdir(cgroupPath); err != nil {
		if errno, ok := err.(syscall.Errno); !ok || errno != syscall.ENOENT {
			return nil, err
		}
	}
	upperdir := filepath.Join(path, "upper")
	if err := os.Mkdir(upperdir, os.ModePerm); err != nil {
		_ = os.RemoveAll(path)
		return nil, err
	}
	workdir := filepath.Join(path, "workdir")
	if err := os.Mkdir(workdir, os.ModePerm); err != nil {
		_ = os.RemoveAll(path)
		return nil, err
	}
	rootfs := filepath.Join(path, "rootfs")
	if err := os.Mkdir(rootfs, os.ModePerm); err != nil {
		_ = os.RemoveAll(path)
		return nil, err
	}
	return &safeexecProcess{
		name:       name,
		path:       path,
		cgroupPath: cgroupPath,
	}, nil
}

func newSafeexecProcessor(path, executionPath, cgroupName string) (*safeexecProcessor, error) {
	cgroupPath, err := getCgroupParentPath()
	if err != nil {
		return nil, err
	}
	if len(cgroupName) != 0 {
		if dir := filepath.Dir(cgroupPath); strings.HasPrefix(dir, "/sys/fs/cgroup") {
			cgroupPath = filepath.Join(dir, cgroupName)
		}
	}
	if err := os.Mkdir(cgroupPath, os.ModePerm); err != nil && !os.IsExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(executionPath, os.ModePerm); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return &safeexecProcessor{
		path:          path,
		executionPath: executionPath,
		cgroupPath:    cgroupPath,
	}, nil
}

func getCgroupParentPath() (string, error) {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := scanner.Text()
		parts := strings.SplitN(text, ":", 3)
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid cgroup line: %q", text)
		}
		if parts[1] == "" {
			return filepath.Join("/sys/fs/cgroup", parts[2]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("cannot find cgroup path")
}
