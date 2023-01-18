package invoker

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
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

type safeexecReport struct {
	Memory   int64
	Time     time.Duration
	ExitCode int
}

func (p *safeexecProcess) Start() error {
	return p.cmd.Start()
}

func (p *safeexecProcess) Wait() (safeexecReport, error) {
	if err := p.cmd.Wait(); err != nil {
		return safeexecReport{}, err
	}
	file, err := os.Open(filepath.Join(p.path, "report.txt"))
	if err != nil {
		return safeexecReport{}, err
	}
	report := safeexecReport{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			return safeexecReport{}, fmt.Errorf("cannot read report")
		}
		switch parts[0] {
		case "memory":
			value, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return safeexecReport{}, fmt.Errorf("cannot parse memory: %w", err)
			}
			report.Memory = value
		case "time":
			value, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return safeexecReport{}, fmt.Errorf("cannot parse time: %w", err)
			}
			report.Time = time.Duration(value) * time.Millisecond
		case "exit_code":
			value, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return safeexecReport{}, fmt.Errorf("cannot parse exit_code: %w", err)
			}
			report.ExitCode = int(value)
		}
	}
	return report, nil
}

// Release releases all associatet resources with process.
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

func (m *safeexecProcessor) Create(ctx context.Context, config safeexecProcessConfig) (*safeexecProcess, error) {
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
	args = append(args, "--report", filepath.Join(process.path, "report.txt"))
	if len(config.Workdir) > 0 {
		args = append(args, "--workdir", config.Workdir)
	}
	args = append(args, config.Command...)
	cmd := exec.CommandContext(ctx, m.path, args...)
	cmd.Stdin = config.Stdin
	cmd.Stdout = config.Stdout
	cmd.Stderr = config.Stderr
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
	if err := setupCgroup(cgroupPath); err != nil {
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

func getCgroupParentPath() (string, error) {
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
			return filepath.Join("/sys/fs/cgroup", parts[2]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("cannot find cgroup path")
}
