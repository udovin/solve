package invoker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gofrs/uuid"
	"github.com/labstack/gommon/log"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"
	"github.com/opencontainers/runc/libcontainer/specconv"
	"golang.org/x/sys/unix"

	"github.com/udovin/solve/pkg"

	_ "github.com/opencontainers/runc/libcontainer/nsenter"
)

type compiler struct {
	Logger            *log.Logger
	Factory           libcontainer.Factory
	ImagePath         string
	CompileArgs       []string
	CompileCwd        string
	CompileEnv        []string
	CompileSourcePath string
	CompileTargetPath string
	CompileLogPath    string
	ExecuteArgs       []string
	ExecuteCwd        string
	ExecuteEnv        []string
	ExecuteBinaryPath string
}

// Compile compiles source file into executable file.
func (c *compiler) Compile(ctx context.Context, source, target, log string) error {
	rootfs, err := makeTempDir()
	if err != nil {
		return err
	}
	if err := pkg.ExtractTarGz(c.ImagePath, rootfs); err != nil {
		return err
	}
	// defer func() {
	// 	_ = os.RemoveAll(rootfs)
	// }()
	sourcePath := filepath.Join(rootfs, c.CompileSourcePath)
	if err := c.copyFileRec(source, sourcePath); err != nil {
		return err
	}
	cwdPath := filepath.Join(rootfs, c.CompileCwd)
	if err := os.MkdirAll(cwdPath, os.ModePerm); err != nil {
		return err
	}
	containerID := "solve-" + filepath.Base(rootfs)
	containerConfig := defaultRootlessConfig(containerID)
	containerConfig.Rootfs = rootfs
	container, err := c.Factory.Create(containerID, containerConfig)
	if err != nil {
		return err
	}
	defer func() {
		_ = container.Destroy()
	}()
	process := libcontainer.Process{
		Args:   c.CompileArgs,
		Env:    c.CompileEnv,
		User:   "root",
		Init:   true,
		Cwd:    c.CompileCwd,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := container.Run(&process); err != nil {
		return fmt.Errorf("unable to start compiler: %w", err)
	}
	{
		state, err := container.OCIState()
		c.Logger.Info("Container error: ", err)
		stateRaw, _ := json.Marshal(state)
		c.Logger.Info("Container state: ", string(stateRaw))
	}
	if state, err := process.Wait(); err != nil {
		return fmt.Errorf("unable to wait compiler: %w", err)
	} else {
		c.Logger.Info("ExitCode: ", state.ExitCode())
		c.Logger.Info("Exited: ", state.Exited())
		c.Logger.Info("String: ", state.String())
	}
	{
		state, err := container.OCIState()
		c.Logger.Info("Container error: ", err)
		stateRaw, _ := json.Marshal(state)
		c.Logger.Info("Container state: ", string(stateRaw))
	}
	if err := c.copyFile(filepath.Join(rootfs, c.CompileLogPath), log); err != nil {
		return fmt.Errorf("unable to copy compile log: %w", err)
	}
	if err := c.copyFile(filepath.Join(rootfs, c.CompileTargetPath), target); err != nil {
		return fmt.Errorf("unable to copy binary: %w", err)
	}
	return nil
}

// Execute executes compiled solution with specified input file.
func (c *compiler) Execute(ctx context.Context, binary, input, output string) error {
	rootfs, err := makeTempDir()
	if err != nil {
		return err
	}
	if err := pkg.ExtractTarGz(c.ImagePath, rootfs); err != nil {
		return err
	}
	// defer func() {
	// 	_ = os.RemoveAll(rootfs)
	// }()
	binaryPath := filepath.Join(rootfs, c.ExecuteBinaryPath)
	if err := c.copyFileRec(binary, binaryPath); err != nil {
		return err
	}
	cwdPath := filepath.Join(rootfs, c.ExecuteCwd)
	if err := os.MkdirAll(cwdPath, os.ModePerm); err != nil {
		return err
	}
	containerID := "solve-" + filepath.Base(rootfs)
	containerConfig := defaultRootlessConfig(containerID)
	containerConfig.Rootfs = rootfs
	container, err := c.Factory.Create(containerID, containerConfig)
	if err != nil {
		return err
	}
	defer func() {
		_ = container.Destroy()
	}()
	process := libcontainer.Process{
		Args:   c.ExecuteArgs,
		Env:    c.ExecuteEnv,
		User:   "root",
		Init:   true,
		Cwd:    c.ExecuteCwd,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := container.Run(&process); err != nil {
		return fmt.Errorf("unable to start compiler: %w", err)
	}
	if _, err := process.Wait(); err != nil {
		return fmt.Errorf("unable to wait compiler: %w", err)
	}
	return nil
}

var defaultEnv = []string{
	"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	"TERM=xterm",
}

func makeTempDir() (string, error) {
	for i := 0; i < 100; i++ {
		name, err := uuid.NewV4()
		if err != nil {
			return "", err
		}
		dirPath := filepath.Join(os.TempDir(), name.String())
		if err := os.MkdirAll(dirPath, 0777); err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", err
		}
		return dirPath, nil
	}
	return "", fmt.Errorf("unable to create temp directory")
}

func (c *compiler) copyFileRec(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
		return err
	}
	return c.copyFile(source, target)
}

func (c *compiler) copyFile(source, target string) error {
	r, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			c.Logger.Warn(err)
		}
	}()
	w, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		if err := w.Close(); err != nil {
			c.Logger.Warn(err)
		}
	}()
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return nil
}

func configDevices() (devices []*devices.Rule) {
	for _, device := range specconv.AllowedDevices {
		devices = append(devices, &device.Rule)
	}
	return devices
}

func defaultRootlessConfig(id string) *configs.Config {
	defaultMountFlags := unix.MS_NOEXEC | unix.MS_NOSUID | unix.MS_NODEV
	caps := []string{
		"CAP_AUDIT_WRITE",
		"CAP_KILL",
		"CAP_NET_BIND_SERVICE",
	}
	// subUIDs, _ := user.CurrentUserSubUIDs()
	// subGIDs, _ := user.CurrentUserSubGIDs()
	return &configs.Config{
		Capabilities: &configs.Capabilities{
			Bounding:    caps,
			Effective:   caps,
			Inheritable: caps,
			Permitted:   caps,
			Ambient:     caps,
		},
		Rlimits: []configs.Rlimit{
			{
				Type: unix.RLIMIT_NOFILE,
				Hard: uint64(1025),
				Soft: uint64(1025),
			},
		},
		RootlessEUID:    true,
		RootlessCgroups: true,
		Cgroups: &configs.Cgroup{
			Name:   id,
			Parent: "system",
			Resources: &configs.Resources{
				MemorySwappiness: nil,
				Devices:          configDevices(),
			},
		},
		Devices:         specconv.AllowedDevices,
		NoNewPrivileges: true,
		NoNewKeyring:    true,
		// NoPivotRoot:     true,
		Readonlyfs: false,
		Hostname:   "runc",
		Mounts: []*configs.Mount{
			{
				Device:      "proc",
				Source:      "proc",
				Destination: "/proc",
				Flags:       defaultMountFlags,
			},
			{
				Device:      "tmpfs",
				Source:      "tmpfs",
				Destination: "/dev",
				Flags:       unix.MS_NOSUID | unix.MS_STRICTATIME,
				Data:        "mode=755,size=65536k",
			},
			{
				Device:      "devpts",
				Source:      "devpts",
				Destination: "/dev/pts",
				Flags:       unix.MS_NOSUID | unix.MS_NOEXEC,
				Data:        "newinstance,ptmxmode=0666,mode=0620",
			},
			{
				Device:      "tmpfs",
				Source:      "shm",
				Destination: "/dev/shm",
				Data:        "mode=1777,size=65536k",
				Flags:       defaultMountFlags,
			},
			{
				Device:      "mqueue",
				Source:      "mqueue",
				Destination: "/dev/mqueue",
				Flags:       defaultMountFlags,
			},
			{
				Device:      "bind",
				Source:      "/sys",
				Destination: "/sys",
				Flags:       defaultMountFlags | unix.MS_RDONLY | unix.MS_BIND | unix.MS_REC,
			},
		},
		Namespaces: configs.Namespaces([]configs.Namespace{
			{Type: configs.NEWNS},
			{Type: configs.NEWPID},
			{Type: configs.NEWIPC},
			{Type: configs.NEWUTS},
			{Type: configs.NEWUSER},
			{Type: configs.NEWCGROUP},
		}),
		UidMappings: []configs.IDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []configs.IDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
		MaskPaths: []string{
			"/proc/acpi",
			"/proc/asound",
			"/proc/kcore",
			"/proc/keys",
			"/proc/latency_stats",
			"/proc/timer_list",
			"/proc/timer_stats",
			"/proc/sched_debug",
			"/sys/firmware",
			"/proc/scsi",
		},
		ReadonlyPaths: []string{
			"/proc/bus",
			"/proc/fs",
			"/proc/irq",
			"/proc/sys",
			"/proc/sysrq-trigger",
		},
	}
}

func init() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runtime.GOMAXPROCS(1)
		runtime.LockOSThread()
		factory, err := libcontainer.New("")
		if err != nil {
			panic(err)
		}
		if err := factory.StartInitialization(); err != nil {
			panic(err)
		}
		panic("--this line should have never been executed, congratulations--")
	}
}
