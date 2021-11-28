package invoker

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gofrs/uuid"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"
	"github.com/opencontainers/runc/libcontainer/specconv"
	"golang.org/x/sys/unix"

	"github.com/udovin/solve/pkg"

	_ "github.com/opencontainers/runc/libcontainer/nsenter"
)

type compiler struct {
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
	if err := copyFileRec(source, sourcePath); err != nil {
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
		Stdout: nil,
		Stderr: nil,
	}
	if err := container.Run(&process); err != nil {
		return fmt.Errorf("unable to start compiler: %w", err)
	}
	if _, err := process.Wait(); err != nil {
		return fmt.Errorf("unable to wait compiler: %w", err)
	}
	if err := copyFile(filepath.Join(rootfs, c.CompileLogPath), log); err != nil {
		return err
	}
	return copyFile(filepath.Join(rootfs, c.CompileTargetPath), target)
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
	if err := copyFileRec(binary, binaryPath); err != nil {
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
		Stdout: nil,
		Stderr: nil,
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

func copyFileRec(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
		return err
	}
	return copyFile(source, target)
}

func copyFile(source, target string) error {
	r, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Println("Error:", err)
		}
	}()
	w, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		if err := w.Close(); err != nil {
			log.Println("Error:", err)
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
				Source:      "proc",
				Destination: "/proc",
				Device:      "proc",
				Flags:       defaultMountFlags,
			},
			{
				Source:      "tmpfs",
				Destination: "/dev",
				Device:      "tmpfs",
				Flags:       unix.MS_NOSUID | unix.MS_STRICTATIME,
				Data:        "mode=755,size=65536k",
			},
			{
				Source:      "devpts",
				Destination: "/dev/pts",
				Device:      "devpts",
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
				Source:      "mqueue",
				Destination: "/dev/mqueue",
				Device:      "mqueue",
				Flags:       defaultMountFlags,
			},
			{
				Source:      "/sys",
				Device:      "bind",
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
