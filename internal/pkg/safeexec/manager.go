package safeexec

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Manager struct {
	path          string
	executionPath string
	cgroupPath    string
	useMemoryPeak bool
	useCPULimit   bool
	pidsLimit     int
}

type ProcessConfig struct {
	TimeLimit   time.Duration
	MemoryLimit int64
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Layers      []string
	Environ     []string
	Workdir     string
	Command     []string
}

const (
	memoryPeakFlag = 1
	cpuLimitFlag   = 2
)

func (m *Manager) Create(ctx context.Context, config ProcessConfig) (*Process, error) {
	workdir := filepath.Clean(string(filepath.Separator) + config.Workdir)
	if !filepath.IsAbs(workdir) {
		// This should never have happened.
		panic(fmt.Errorf("path %q is not absolute", workdir))
	}
	process, err := m.prepareProcess()
	if err != nil {
		return nil, err
	}
	var args []string
	args = append(args, "--time-limit", fmt.Sprint(config.TimeLimit.Milliseconds()))
	args = append(args, "--memory-limit", fmt.Sprint(config.MemoryLimit))
	if m.useCPULimit {
		args = append(args, "--cpu-limit", fmt.Sprint(100))
	}
	if m.pidsLimit > 0 {
		args = append(args, "--pids-limit", fmt.Sprint(m.pidsLimit))
	}
	args = append(args, "--overlay-lowerdir", strings.Join(config.Layers, ":"))
	args = append(args, "--overlay-upperdir", filepath.Join(process.path, "upper"))
	args = append(args, "--overlay-workdir", filepath.Join(process.path, "workdir"))
	args = append(args, "--rootfs", filepath.Join(process.path, "rootfs"))
	args = append(args, "--cgroup-path", process.cgroupPath)
	args = append(args, "--report", filepath.Join(process.path, "report.txt"))
	args = append(args, "--workdir", workdir)
	var flags int
	if m.useMemoryPeak {
		flags = flags | memoryPeakFlag
	}
	if m.useCPULimit {
		flags = flags | cpuLimitFlag
	}
	if flags > 0 {
		args = append(args, "--flags", fmt.Sprint(flags))
	}
	for _, env := range config.Environ {
		args = append(args, "--env", env)
	}
	args = append(args, config.Command...)
	cmd := exec.CommandContext(ctx, m.path, args...)
	cmd.Stdin = config.Stdin
	cmd.Stdout = config.Stdout
	cmd.Stderr = config.Stderr
	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = time.Second
	process.workdir = workdir
	process.cmd = cmd
	return process, nil
}

func (m *Manager) HasMemoryPeak() bool {
	return m.useMemoryPeak
}

func (m *Manager) HasCPULimit() bool {
	return m.useCPULimit
}

func (m *Manager) createProcessName() (string, error) {
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

func (m *Manager) prepareProcess() (*Process, error) {
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
	return &Process{
		name:       name,
		path:       path,
		cgroupPath: cgroupPath,
	}, nil
}

type Option func(*Manager) error

func WithDisableMemoryPeak(m *Manager) error {
	m.useMemoryPeak = false
	return nil
}

func WithDisableCpuLimit(m *Manager) error {
	m.useCPULimit = false
	return nil
}

func WithPidsLimit(limit int) Option {
	return func(m *Manager) error {
		m.pidsLimit = limit
		return nil
	}
}

func NewManager(path, executionPath, cgroupName string, options ...Option) (*Manager, error) {
	cgroupPath, err := getCurrentCgroupPath()
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(cgroupName, "/") {
		cgroupPath = cgroupRootPath
	}
	cgroupPath = filepath.Join(cgroupPath, cgroupName)
	if !strings.HasPrefix(cgroupPath, cgroupRootPath) {
		return nil, fmt.Errorf("invalid cgroup path: %s", cgroupPath)
	}
	if cgroupPath == cgroupRootPath {
		return nil, fmt.Errorf("cannot use root cgroup")
	}
	m := Manager{
		path:          path,
		executionPath: executionPath,
		cgroupPath:    cgroupPath,
		useMemoryPeak: true,
		useCPULimit:   true,
	}
	for _, option := range options {
		if err := option(&m); err != nil {
			return nil, err
		}
	}
	if err := setupCgroup(m.cgroupPath); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(m.executionPath, os.ModePerm); err != nil && !os.IsExist(err) {
		return nil, err
	}
	if err := fixCgroupFeature(&m.useMemoryPeak, m.cgroupPath, "memory.peak"); err != nil {
		return nil, err
	}
	if err := fixCgroupFeature(&m.useCPULimit, m.cgroupPath, "cpu.max"); err != nil {
		return nil, err
	}
	return &m, nil
}

func setupCgroup(path string) error {
	if err := os.Mkdir(path, os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}
	file, err := os.Open(filepath.Join(path, "cgroup.controllers"))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	subtreeFile, err := os.OpenFile(filepath.Join(path, "cgroup.subtree_control"), os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() { _ = subtreeFile.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		parts := strings.Split(text, " ")
		for _, part := range parts {
			_, err = subtreeFile.WriteString(fmt.Sprintf("+%s", part))
			if err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func fixCgroupFeature(enabled *bool, cgroupPath, name string) error {
	if !*enabled {
		return nil
	}
	if _, err := os.Stat(filepath.Join(cgroupPath, name)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			*enabled = false
			return nil
		}
		return err
	}
	return nil
}

const cgroupRootPath = "/sys/fs/cgroup"

func getCurrentCgroupPath() (string, error) {
	file, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		parts := strings.SplitN(text, ":", 3)
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid cgroup line: %q", text)
		}
		if parts[1] == "" {
			return filepath.Join(cgroupRootPath, parts[2]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("cannot find cgroup path")
}
