package invoker

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"
	"github.com/opencontainers/runc/libcontainer/specconv"
	"github.com/sirupsen/logrus"
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
	logrus.SetLevel(logrus.FatalLevel)
}

type processConfig struct {
	Args   []string
	Dir    string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type process struct {
	config  processConfig
	process *libcontainer.Process
}

func (p *process) Signal(signal os.Signal) error {
	return p.process.Signal(signal)
}

func (p *process) Wait() (*os.ProcessState, error) {
	return p.process.Wait()
}

type containerConfig struct {
	Layers      []string
	Init        processConfig
	MemoryLimit int64
}

type container struct {
	config    containerConfig
	container libcontainer.Container
	upperDir  string
}

func (c *container) ID() string {
	return c.container.ID()
}

func (c *container) GetUpperDir() string {
	return c.upperDir
}

func (c *container) Start() (*process, error) {
	p := c.buildProcess(c.config.Init)
	p.Init = true
	if err := c.container.Run(&p); err != nil {
		return nil, err
	}
	return &process{config: c.config.Init, process: &p}, nil
}

func (c *container) Signal(signal os.Signal) error {
	return c.container.Signal(signal, true)
}

func (c *container) Destroy() error {
	return c.container.Destroy()
}

func (c *container) buildProcess(config processConfig) libcontainer.Process {
	process := libcontainer.Process{
		Args:   config.Args,
		Cwd:    config.Dir,
		Env:    config.Env,
		User:   "0:0",
		Stdin:  config.Stdin,
		Stdout: config.Stdout,
		Stderr: config.Stderr,
	}
	return process
}

type factory struct {
	factory libcontainer.Factory
	dir     string
}

func newFactory(dir string) (*factory, error) {
	f, err := libcontainer.New(
		filepath.Join(dir, "state"),
		libcontainer.InitArgs(os.Args[0], "init"),
	)
	if err != nil {
		return nil, err
	}
	return &factory{factory: f, dir: dir}, nil
}

func (f *factory) Create(config containerConfig) (*container, error) {
	id, err := generateID()
	if err != nil {
		return nil, err
	}
	containerPath := filepath.Join(f.dir, "containers", id)
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
	lowerPath := strings.Join(config.Layers, ":")
	defaultMountFlags := unix.MS_NOEXEC | unix.MS_NOSUID | unix.MS_NODEV
	defaultCapabilities := []string{
		"CAP_AUDIT_WRITE",
		"CAP_KILL",
		"CAP_NET_BIND_SERVICE",
	}
	mounts := []*configs.Mount{
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
			Flags:       defaultMountFlags,
			Data:        "mode=1777,size=65536k",
		},
		{
			Device:      "mqueue",
			Source:      "mqueue",
			Destination: "/dev/mqueue",
			Flags:       defaultMountFlags,
		},
		{
			Device:      "sysfs",
			Source:      "sysfs",
			Destination: "/sys",
			Flags:       defaultMountFlags | unix.MS_RDONLY,
		},
		{
			Device:      "cgroup",
			Source:      "cgroup",
			Destination: "/sys/fs/cgroup",
			Flags:       defaultMountFlags | unix.MS_RELATIME | unix.MS_RDONLY,
		},
	}
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
			Name:   id,
			Parent: "solve",
			Resources: &configs.Resources{
				Devices:           configDevices(),
				Memory:            config.MemoryLimit,
				MemorySwap:        config.MemoryLimit,
				MemoryReservation: config.MemoryLimit,
			},
			Rootless: true,
		},
		Capabilities: &configs.Capabilities{
			Bounding:    defaultCapabilities,
			Effective:   defaultCapabilities,
			Inheritable: defaultCapabilities,
			Permitted:   defaultCapabilities,
			Ambient:     defaultCapabilities,
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
		Mounts: mounts,
		UidMappings: []configs.IDMap{
			{ContainerID: 0, HostID: os.Geteuid(), Size: 1},
		},
		GidMappings: []configs.IDMap{
			{ContainerID: 0, HostID: os.Getegid(), Size: 1},
		},
		Networks: []*configs.Network{
			{Type: "loopback", Address: "127.0.0.1/0", Gateway: "localhost"},
		},
		Rlimits: []configs.Rlimit{
			{
				Type: unix.RLIMIT_NOFILE,
				Hard: 1024,
				Soft: 1024,
			},
			// {
			// 	Type: unix.RLIMIT_AS,
			// 	Hard: uint64(config.MemoryLimit),
			// 	Soft: uint64(config.MemoryLimit),
			// },
		},
	}
	c, err := f.factory.Create(id, &containerConfig)
	if err != nil {
		return nil, err
	}
	return &container{
		config:    config,
		container: c,
		upperDir:  upperPath,
	}, nil
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
