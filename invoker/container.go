package invoker

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"
	"github.com/opencontainers/runc/libcontainer/specconv"
	"github.com/opencontainers/runc/libcontainer/user"
	"golang.org/x/sys/unix"

	_ "github.com/opencontainers/runc/libcontainer/nsenter"
)

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

type ProcessConfig struct {
	Args []string
}

type Process struct {
	config  ProcessConfig
	process *libcontainer.Process
}

func (p *Process) Signal(signal os.Signal) error {
	return p.process.Signal(signal)
}

func (p *Process) Wait() (*os.ProcessState, error) {
	return p.process.Wait()
}

type ContainerConfig struct {
	Layers []string
	Init   ProcessConfig
}

type Container struct {
	config    ContainerConfig
	container libcontainer.Container
}

func (c *Container) ID() string {
	return c.container.ID()
}

func (c *Container) Start() (*Process, error) {
	process := c.buildProcess(c.config.Init)
	process.Init = true
	if err := c.container.Run(&process); err != nil {
		return nil, err
	}
	return &Process{config: c.config.Init, process: &process}, nil
}

func (c *Container) Signal(signal os.Signal) error {
	return c.container.Signal(signal, true)
}

func (c *Container) Destroy() error {
	return c.container.Destroy()
}

func (c *Container) buildProcess(config ProcessConfig) libcontainer.Process {
	process := libcontainer.Process{
		Args: config.Args,
	}
	return process
}

type Processor struct {
	factory libcontainer.Factory
	dir     string
}

func (p *Processor) Create(config ContainerConfig) (*Container, error) {
	id, err := generateID()
	if err != nil {
		return nil, err
	}
	containerPath := filepath.Join(p.dir, "containers", id)
	if err := os.MkdirAll(containerPath, os.ModePerm); err != nil {
		return nil, err
	}
	rootfsPath := filepath.Join(containerPath, "rootfs")
	if err := os.Mkdir(rootfsPath, os.ModePerm); err != nil {
		return nil, err
	}
	workPath := filepath.Join(containerPath, "work")
	if err := os.Mkdir(workPath, os.ModePerm); err != nil {
		return nil, err
	}
	upperPath := filepath.Join(containerPath, "upper")
	if err := os.Mkdir(upperPath, os.ModePerm); err != nil {
		return nil, err
	}
	uidMappings, err := getUIDMappings()
	if err != nil {
		return nil, err
	}
	gidMappings, err := getGIDMappings()
	if err != nil {
		return nil, err
	}
	lowerPath := strings.Join(config.Layers, ":")
	defaultMountFlags := unix.MS_NOEXEC | unix.MS_NOSUID | unix.MS_NODEV
	containerConfig := configs.Config{
		Hostname:        id,
		Rootfs:          rootfsPath,
		RootlessEUID:    true,
		RootlessCgroups: true,
		NoNewPrivileges: true,
		NoNewKeyring:    true,
		Namespaces: configs.Namespaces([]configs.Namespace{
			{Type: configs.NEWNS},
			{Type: configs.NEWUTS},
			{Type: configs.NEWIPC},
			{Type: configs.NEWPID},
			{Type: configs.NEWUSER},
			{Type: configs.NEWNET},
			{Type: configs.NEWCGROUP},
		}),
		Devices: specconv.AllowedDevices,
		Cgroups: &configs.Cgroup{
			Name:   "c-" + id,
			Parent: "system",
			Resources: &configs.Resources{
				MemorySwappiness: nil,
				Devices:          configDevices(),
			},
			Rootless: true,
		},
		Capabilities: &configs.Capabilities{
			Bounding: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SETFCAP",
				"CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE",
				"CAP_SYS_CHROOT",
				"CAP_KILL",
				"CAP_AUDIT_WRITE",
			},
			Effective: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SETFCAP",
				"CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE",
				"CAP_SYS_CHROOT",
				"CAP_KILL",
				"CAP_AUDIT_WRITE",
			},
			Permitted: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SETFCAP",
				"CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE",
				"CAP_SYS_CHROOT",
				"CAP_KILL",
				"CAP_AUDIT_WRITE",
			},
			Ambient: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SETFCAP",
				"CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE",
				"CAP_SYS_CHROOT",
				"CAP_KILL",
				"CAP_AUDIT_WRITE",
			},
		},
		MaskPaths: []string{
			"/proc/kcore",
			"/sys/firmware",
		},
		ReadonlyPaths: []string{
			"/proc/sys", "/proc/sysrq-trigger", "/proc/irq", "/proc/bus",
		},
		Mounts: []*configs.Mount{
			{
				Device:      "overlay",
				Source:      "overlay",
				Destination: "/",
				Data: fmt.Sprintf(
					"lowerdir=%s,upperdir=%s,workdir=%s",
					lowerPath, upperPath, workPath,
				),
			},
			{
				Device:      "proc",
				Source:      "proc",
				Destination: "/proc",
				Flags:       defaultMountFlags,
			},
			{
				Source:      "tmpfs",
				Destination: "/dev",
				Device:      "tmpfs",
				Flags:       unix.MS_NOSUID | unix.MS_STRICTATIME,
				Data:        "mode=755",
			},
			// {
			// 	Source:      "devpts",
			// 	Destination: "/dev/pts",
			// 	Device:      "devpts",
			// 	Flags:       unix.MS_NOSUID | unix.MS_NOEXEC,
			// 	Data:        "newinstance,ptmxmode=0666,mode=0620,gid=5",
			// },
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
				Source:      "sysfs",
				Destination: "/sys",
				Device:      "sysfs",
				Flags:       defaultMountFlags | unix.MS_RDONLY,
			},
		},
		UidMappings: uidMappings,
		GidMappings: gidMappings,
		Networks: []*configs.Network{
			{
				Type:    "loopback",
				Address: "127.0.0.1/0",
				Gateway: "localhost",
			},
		},
		Rlimits: []configs.Rlimit{
			{
				Type: unix.RLIMIT_NOFILE,
				Hard: uint64(1025),
				Soft: uint64(1025),
			},
		},
	}
	container, err := p.factory.Create(id, &containerConfig)
	if err != nil {
		return nil, err
	}
	return &Container{config: config, container: container}, nil
}

func configDevices() (devices []*devices.Rule) {
	for _, device := range specconv.AllowedDevices {
		devices = append(devices, &device.Rule)
	}
	return devices
}

func generateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func getUIDMappings() ([]configs.IDMap, error) {
	subUIDs, err := user.CurrentUserSubUIDs()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	mappings := []configs.IDMap{
		{ContainerID: 0, HostID: os.Getuid(), Size: 1},
	}
	if len(subUIDs) > 0 {
		// mappings = append(mappings, configs.IDMap{
		// 	ContainerID: 1,
		// 	HostID:      int(subUIDs[0].SubID),
		// 	Size:        int(subUIDs[0].Count),
		// })
	}
	return mappings, nil
}

func getGIDMappings() ([]configs.IDMap, error) {
	subGIDs, err := user.CurrentUserSubGIDs()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	mappings := []configs.IDMap{
		{ContainerID: 0, HostID: os.Getgid(), Size: 1},
	}
	if len(subGIDs) > 0 {
		// mappings = append(mappings, configs.IDMap{
		// 	ContainerID: 1,
		// 	HostID:      int(subGIDs[0].SubID),
		// 	Size:        int(subGIDs[0].Count),
		// })
	}
	return mappings, nil
}
