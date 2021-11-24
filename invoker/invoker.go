package invoker

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"
	"github.com/opencontainers/runc/libcontainer/specconv"
	"golang.org/x/sys/unix"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg"

	_ "github.com/opencontainers/runc/libcontainer/nsenter"
)

// Invoker represents manager for asynchronous actions (invocations).
type Invoker struct {
	core    *core.Core
	factory libcontainer.Factory
}

// New creates a new instance of Invoker.
func New(c *core.Core) *Invoker {
	return &Invoker{core: c}
}

// Start starts invoker daemons.
//
// This function will spawn config.Invoker.Threads amount of goroutines.
func (s *Invoker) Start() error {
	if s.factory != nil {
		return fmt.Errorf("factory already created")
	}
	factory, err := libcontainer.New(
		"/tmp/libcontainer",
		libcontainer.RootlessCgroupfs,
		libcontainer.InitArgs(os.Args[0], "init"),
	)
	if err != nil {
		return err
	}
	s.factory = factory
	threads := s.core.Config.Invoker.Threads
	if threads <= 0 {
		threads = 1
	}
	for i := 0; i < threads; i++ {
		s.core.StartTask(s.runDaemon)
	}
	return nil
}

func (s *Invoker) runDaemon(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if ok := s.runDaemonTick(ctx); !ok {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}
	}
}

func (s *Invoker) runDaemonTick(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	var task models.Task
	if err := s.core.WithTx(ctx, func(tx *sql.Tx) error {
		var err error
		task, err = s.core.Tasks.PopQueuedTx(tx)
		return err
	}); err != nil {
		if err != sql.ErrNoRows {
			s.core.Logger().Error("Error:", err)
		}
		return false
	}
	defer func() {
		if r := recover(); r != nil {
			task.Status = models.Failed
			s.core.Logger().Error("Task panic:", r)
			panic(r)
		}
		ctx, cancel := context.WithDeadline(context.Background(), time.Unix(task.ExpireTime, 0))
		defer cancel()
		if err := s.core.WithTx(ctx, func(tx *sql.Tx) error {
			return s.core.Tasks.UpdateTx(tx, task)
		}); err != nil {
			s.core.Logger().Error("Error:", err)
		}
	}()
	var waiter sync.WaitGroup
	defer waiter.Wait()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	waiter.Add(1)
	go func() {
		defer waiter.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				select {
				case <-ctx.Done():
					return
				default:
				}
				if time.Now().After(time.Unix(task.ExpireTime, 0)) {
					s.core.Logger().Error("Task expired:", task.ID)
					return
				}
				clone := task
				if err := s.core.WithTx(ctx, func(tx *sql.Tx) error {
					clone.ExpireTime = time.Now().Add(5 * time.Second).Unix()
					return s.core.Tasks.UpdateTx(tx, clone)
				}); err != nil {
					s.core.Logger().Warn("Unable to ping task:", err)
				} else {
					task.ExpireTime = clone.ExpireTime
				}
			}
		}
	}()
	err := s.onTask(ctx, task)
	cancel()
	waiter.Wait()
	if err != nil {
		s.core.Logger().Error("Task failed: ", err)
		task.Status = models.Failed
	} else {
		task.Status = models.Succeeded
	}
	return true
}

func (s *Invoker) onTask(ctx context.Context, task models.Task) error {
	s.core.Logger().Debug("Received new task: ", task.ID)
	switch task.Kind {
	case models.JudgeSolution:
		return s.onJudgeSolution(ctx, task)
	default:
		s.core.Logger().Error("Unknown task: ", task.Kind)
		return fmt.Errorf("unknown task")
	}
}

func (s *Invoker) onJudgeSolution(ctx context.Context, task models.Task) error {
	var taskConfig models.JudgeSolutionConfig
	if err := task.ScanConfig(&taskConfig); err != nil {
		return fmt.Errorf("unable to scan task config: %w", err)
	}
	solution, err := s.core.Solutions.Get(taskConfig.SolutionID)
	if err != nil {
		return fmt.Errorf("unable to fetch task solution: %w", err)
	}
	problem, err := s.core.Problems.Get(solution.ProblemID)
	if err != nil {
		return fmt.Errorf("unable to fetch task problem: %w", err)
	}
	tempDir, err := makeTempDir()
	if err != nil {
		return err
	}
	s.core.Logger().Info(tempDir)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()
	problemPath := path.Join(
		s.core.Config.Storage.ProblemsDir,
		fmt.Sprintf("%d.zip", problem.ID),
	)
	tempProblemPath := path.Join(tempDir, "problem")
	if err := pkg.ExtractZip(problemPath, tempProblemPath); err != nil {
		return err
	}
	compierPath := path.Join(
		s.core.Config.Storage.CompilersDir,
		"dosbox-tasm.tar.gz",
	)
	solutionPath := path.Join(
		s.core.Config.Storage.SolutionsDir,
		fmt.Sprintf("%d.txt", solution.ID),
	)
	tempSolutionPath := path.Join(tempDir, "solution")
	{
		rootfs := path.Join(tempDir, "rootfs-compile")
		if err := pkg.ExtractTarGz(compierPath, rootfs); err != nil {
			return err
		}
		defer func() {
			_ = os.RemoveAll(rootfs)
		}()
		if err := os.MkdirAll(path.Join(rootfs, "home", "solution"), os.ModePerm); err != nil {
			return err
		}
		if err := copyFile(solutionPath, path.Join(rootfs, "home", "solution", "solution.asm")); err != nil {
			return err
		}
		containerID := fmt.Sprintf("task-%d-compile", task.ID)
		containerConfig := defaultRootlessConfig(containerID)
		containerConfig.Rootfs = rootfs
		container, err := s.factory.Create(containerID, containerConfig)
		if err != nil {
			return err
		}
		defer func() {
			_ = container.Destroy()
		}()
		process := libcontainer.Process{
			Args: []string{"dosbox", "-conf", "/dosbox_compile.conf"},
			Env: []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"TERM=xterm",
			},
			User:   "root",
			Init:   true,
			Cwd:    "/",
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		if err := container.Run(&process); err != nil {
			return err
		}
		if _, err := process.Wait(); err != nil {
			return fmt.Errorf("unable to wait compiler: %w", err)
		}
		if err := copyFile(path.Join(rootfs, "home", "solution", "SOLUTION.EXE"), tempSolutionPath); err != nil {
			return fmt.Errorf("unable to fetch solution: %w", err)
		}
	}
	return fmt.Errorf("not implemented")
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

		// https://github.com/opencontainers/runc/issues/1456#issuecomment-303784735
		NoNewKeyring: true,
		NoPivotRoot:  true,

		// Rootfs:          chrootDir,
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

func makeTempDir() (string, error) {
	for i := 0; i < 100; i++ {
		name, err := uuid.NewV4()
		if err != nil {
			return "", err
		}
		dirPath := path.Join(os.TempDir(), name.String())
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
