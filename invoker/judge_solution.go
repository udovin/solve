package invoker

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/specconv"
	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg"
	"golang.org/x/sys/unix"
)

func init() {
	registerTaskImpl(models.JudgeSolutionTask, &judgeSolutionTask{})
}

type judgeSolutionTask struct {
	invoker      *Invoker
	config       models.JudgeSolutionTaskConfig
	solution     models.Solution
	problem      models.Problem
	compiler     models.Compiler
	tempDir      string
	problemPath  string
	compilerPath string
	solutionPath string
}

func (judgeSolutionTask) New(invoker *Invoker) taskImpl {
	return &judgeSolutionTask{invoker: invoker}
}

func (t *judgeSolutionTask) Execute(ctx TaskContext) error {
	// Fetch information about task.
	if err := ctx.ScanConfig(&t.config); err != nil {
		return fmt.Errorf("unable to scan task config: %w", err)
	}
	solution, err := t.invoker.getSolution(ctx, t.config.SolutionID)
	if err != nil {
		return fmt.Errorf("unable to fetch task solution: %w", err)
	}
	problem, err := t.invoker.core.Problems.Get(solution.ProblemID)
	if err != nil {
		return fmt.Errorf("unable to fetch task problem: %w", err)
	}
	compiler, err := t.invoker.core.Compilers.Get(solution.CompilerID)
	if err != nil {
		return fmt.Errorf("unable to fetch task compiler: %w", err)
	}
	t.solution = solution
	t.problem = problem
	t.compiler = compiler
	return t.executeImpl(ctx)
}

func (t *judgeSolutionTask) prepareProblem(ctx TaskContext) error {
	problemFile, err := t.invoker.files.DownloadFile(ctx, t.problem.PackageID)
	if err != nil {
		return fmt.Errorf("cannot download problem: %w", err)
	}
	defer problemFile.Close()
	tempProblemPath := filepath.Join(t.tempDir, "problem")
	if err := pkg.ExtractZip(problemFile.Name(), tempProblemPath); err != nil {
		return fmt.Errorf("cannot extract problem: %w", err)
	}
	t.problemPath = tempProblemPath
	return nil
}

func (t *judgeSolutionTask) prepareCompiler(ctx TaskContext) error {
	compilerFile, err := t.invoker.files.DownloadFile(ctx, t.compiler.ImageID)
	if err != nil {
		return fmt.Errorf("cannot download rootfs: %w", err)
	}
	defer compilerFile.Close()
	tempCompilerPath := filepath.Join(t.tempDir, "compiler")
	if err := pkg.ExtractTarGz(compilerFile.Name(), tempCompilerPath); err != nil {
		return fmt.Errorf("cannot extract rootfs: %w", err)
	}
	t.compilerPath = tempCompilerPath
	return nil
}

func (t *judgeSolutionTask) prepareSolution(ctx TaskContext) error {
	if t.solution.ContentID == 0 {
		tempSolutionPath := filepath.Join(t.tempDir, "solution.txt")
		err := ioutil.WriteFile(tempSolutionPath, []byte(t.solution.Content), fs.ModePerm)
		if err != nil {
			return fmt.Errorf("cannot write solution: %w", err)
		}
		t.solutionPath = tempSolutionPath
		return nil
	}
	solutionFile, err := t.invoker.files.DownloadFile(ctx, int64(t.solution.ContentID))
	if err != nil {
		return fmt.Errorf("cannot download solution: %w", err)
	}
	defer solutionFile.Close()
	tempSolutionPath := filepath.Join(t.tempDir, "solution.bin")
	file, err := os.Create(tempSolutionPath)
	if err != nil {
		return fmt.Errorf("cannot create solution: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(file, solutionFile); err != nil {
		return fmt.Errorf("cannot write solution: %w", err)
	}
	t.solutionPath = tempSolutionPath
	return nil
}

func (t *judgeSolutionTask) compileSolution(ctx TaskContext) (bool, error) {
	config, err := t.compiler.GetConfig()
	if err != nil {
		return false, err
	}
	layersDir := filepath.Join(t.tempDir, "layers")
	if err := os.Mkdir(layersDir, os.ModePerm); err != nil {
		return false, err
	}
	upperDir := filepath.Join(layersDir, "upper")
	if err := os.Mkdir(upperDir, os.ModePerm); err != nil {
		return false, err
	}
	workDir := filepath.Join(layersDir, "work")
	if err := os.Mkdir(workDir, os.ModePerm); err != nil {
		return false, err
	}
	mergeDir := filepath.Join(layersDir, "merge")
	if err := os.Mkdir(mergeDir, os.ModePerm); err != nil {
		return false, err
	}
	if source := config.Compile.Source; source != nil {
		if strings.Contains(*source, "..") {
			return false, fmt.Errorf("illegal file path: %q", *source)
		}
		sourcePath := filepath.Join(upperDir, *source)
		if err := copyFileRec(t.solutionPath, sourcePath); err != nil {
			return false, err
		}
	}
	defaultMountFlags := unix.MS_NOEXEC | unix.MS_NOSUID | unix.MS_NODEV
	caps := []string{
		"CAP_AUDIT_WRITE",
		"CAP_KILL",
		"CAP_NET_BIND_SERVICE",
	}
	containerID := "solve-" + filepath.Base(t.tempDir)
	containerConfig := configs.Config{
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
			Name:   containerID,
			Parent: "system",
			Resources: &configs.Resources{
				MemorySwappiness: nil,
				Devices:          configDevices(),
			},
			Rootless: true,
		},
		Devices:         specconv.AllowedDevices,
		NoNewPrivileges: true,
		NoNewKeyring:    true,
		NoPivotRoot:     false,
		Readonlyfs:      false,
		Hostname:        "compiler",
		Rootfs:          mergeDir,
		Mounts: []*configs.Mount{
			{
				Device:      "overlay",
				Source:      "overlay",
				Destination: "/",
				Data:        fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", t.compilerPath, upperDir, workDir),
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
	container, err := t.invoker.factory.Create(containerID, &containerConfig)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = container.Destroy()
	}()
	process := libcontainer.Process{
		Args:   strings.Fields(config.Compile.Command),
		Env:    config.Compile.Environ,
		User:   "root",
		Init:   true,
		Cwd:    config.Compile.Workdir,
		Stdout: nil,
		Stderr: nil,
	}
	if err := container.Run(&process); err != nil {
		return false, fmt.Errorf("unable to start compiler: %w", err)
	}
	state, err := process.Wait()
	if err != nil {
		return false, fmt.Errorf("unable to wait compiler: %w", err)
	}
	if state.ExitCode() != 0 {
		return false, nil
	}
	return true, nil
}

func (t *judgeSolutionTask) executeImpl(ctx TaskContext) error {
	tempDir, err := makeTempDir()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()
	t.tempDir = tempDir
	if err := t.prepareProblem(ctx); err != nil {
		return fmt.Errorf("cannot prepare problem: %w", err)
	}
	if err := t.prepareCompiler(ctx); err != nil {
		return fmt.Errorf("cannot prepare compiler: %w", err)
	}
	if err := t.prepareSolution(ctx); err != nil {
		return fmt.Errorf("cannot prepare solution: %w", err)
	}
	if ok, err := t.compileSolution(ctx); err != nil {
		return fmt.Errorf("cannot compile solution: %w", err)
	} else if !ok {
		return nil
	}
	return fmt.Errorf("not implemented")
}
