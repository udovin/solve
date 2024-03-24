#define _GNU_SOURCE
#include <linux/sched.h>
#include <sched.h>
#include <signal.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <time.h>
#include <sys/mount.h>
#include <sys/resource.h>
#include <sys/sendfile.h>
#include <sys/stat.h>
#include <sys/syscall.h>
#include <sys/wait.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <limits.h>

#define STACK_SIZE 8096
#define OVERLAY_DATA "lowerdir=%s,upperdir=%s,workdir=%s"
#define PROC_PATH "/proc"
#define CGROUP_PROCS_FILE "cgroup.procs"
#define CGROUP_MEMORY_MAX_FILE "memory.max"
#define CGROUP_PIDS_MAX_FILE "pids.max"
#define CGROUP_MEMORY_SWAP_MAX_FILE "memory.swap.max"
#define CGROUP_MEMORY_CURRENT_FILE "memory.current"
#define CGROUP_MEMORY_PEAK_FILE "memory.peak"
#define CGROUP_CPU_MAX_FILE "cpu.max"
#define CGROUP_MEMORY_EVENTS_FILE "memory.events"
#define CGROUP_CPU_STAT_FILE "cpu.stat"

#define MEMORY_PEAK_FLAG 1
#define CPU_LIMIT_FLAG 2

typedef struct {
	char* rootfs;
	char* overlayLowerdir;
	char* overlayUpperdir;
	char* overlayWorkdir;
	char* workdir;
	char** args;
	int argsLen;
	char** environ;
	int environLen;
	char* cgroupPath;
	int memoryLimit; // Bytes.
	int timeLimit;   // Milliseconds.
	int cpuLimit;    // Percent.
	int pidsLimit;   // PIDS amount.
	int flags;
	char* report;
	int initializePipe[2];
	int finalizePipe[2];
} Context;

static inline void ensure(int value, const char* message) {
	if (!value) {
		puts(message);
		exit(EXIT_FAILURE);
	}
}

static inline void setupOverlayfs(const Context* ctx) {
	char* data = malloc((strlen(ctx->overlayLowerdir) + strlen(ctx->overlayUpperdir) + strlen(ctx->overlayWorkdir) + strlen(OVERLAY_DATA)) * sizeof(char));
	ensure(data != 0, "cannot allocate rootfs overlay data");
	sprintf(data, OVERLAY_DATA, ctx->overlayLowerdir, ctx->overlayUpperdir, ctx->overlayWorkdir);
	ensure(mount("overlay", ctx->rootfs, "overlay", 0, data) == 0, "cannot mount rootfs overlay");
	free(data);
}

static inline void mkdirAll(int prefix, char* path) {
	for (int i = prefix; path[i] != 0; ++i) {
		if (path[i] == '/' && i > prefix) {
			path[i] = 0;
			if (mkdir(path, 0755) != 0) {
				ensure(errno == EEXIST, "cannot create directory");
			}
			path[i] = '/';
		}
	}
	if (mkdir(path, 0755) != 0) {
		ensure(errno == EEXIST, "cannot create directory");
	}
}

static inline void setupMount(const Context* ctx, const char* source, const char* target, const char* device, unsigned long flags, const void* data) {
	char* path = malloc((strlen(ctx->rootfs) + strlen(target) + 1) * sizeof(char));
	ensure(path != 0, "cannot allocate");
	strcpy(path, ctx->rootfs);
	strcat(path, target);
	mkdirAll(strlen(ctx->rootfs), path);
	ensure(mount(source, path, device, flags, data) == 0, "cannot mount");
	free(path);
}

static inline void createDev(int prefix, char* path) {
	for (int i = prefix; path[i] != 0; ++i) {
		if (path[i] == '/' && i > prefix) {
			path[i] = 0;
			if (mkdir(path, 0755) != 0) {
				ensure(errno == EEXIST, "cannot create directory");
			}
			path[i] = '/';
		}
	}
	int fd = open(path, O_CREAT, 0000);
	ensure(fd != -1, "cannot create file");
	close(fd);
}

static inline void setupDevMount(const Context* ctx, const char* source, const char* target) {
	char* path = malloc((strlen(ctx->rootfs) + strlen(target) + 1) * sizeof(char));
	ensure(path != 0, "cannot allocate");
	strcpy(path, ctx->rootfs);
	strcat(path, target);
	createDev(strlen(ctx->rootfs), path);
	ensure(mount(source, path, NULL, MS_BIND, NULL) == 0, "cannot mount");
	free(path);
}

static inline void pivotRoot(const Context* ctx) {
	int oldroot = open("/", O_DIRECTORY | O_RDONLY);
	ensure(oldroot != -1, "cannot open old root");
	int newroot = open(ctx->rootfs, O_DIRECTORY | O_RDONLY);
	ensure(newroot != -1, "cannot open new root");
	ensure(fchdir(newroot) == 0, "cannot chdir to new root");
	ensure(syscall(SYS_pivot_root, ".", ".") == 0, "cannot pivot root");
	close(newroot);
	ensure(fchdir(oldroot) == 0, "cannot chdir to new old");
	ensure(mount(NULL, ".", NULL, MS_SLAVE | MS_REC, NULL) == 0, "cannot remount old root");
	ensure(umount2(".", MNT_DETACH) == 0, "cannot unmount old root");
	close(oldroot);
	ensure(chdir("/") == 0, "cannot chdir to \"/\"");
}

static inline void setupUserNamespace(const Context* ctx) {
	// We should wait for setup of user namespace from parent.
	char c;
	ensure(read(ctx->initializePipe[0], &c, 1) == 0, "cannot wait initialize pipe to close");
	close(ctx->initializePipe[0]);
}

static inline void setupMountNamespace(const Context* ctx) {
	// First of all make all changes are private for current root.
	ensure(mount(NULL, "/", NULL, MS_SLAVE | MS_REC, NULL) == 0, "cannot remount \"/\"");
	ensure(mount(NULL, "/", NULL, MS_PRIVATE, NULL) == 0, "cannot remount \"/\"");
	ensure(mount(ctx->rootfs, ctx->rootfs, "bind", MS_BIND | MS_REC, NULL) == 0, "cannot remount rootfs");
	setupOverlayfs(ctx);
	setupMount(ctx, "sysfs", "/sys", "sysfs", MS_NOEXEC | MS_NOSUID | MS_NODEV | MS_RDONLY, NULL);
	setupMount(ctx, "proc", PROC_PATH, "proc", MS_NOEXEC | MS_NOSUID | MS_NODEV, NULL);
	setupMount(ctx, "tmpfs", "/dev", "tmpfs", MS_NOSUID | MS_STRICTATIME, "mode=755,size=65536k");
	setupMount(ctx, "devpts", "/dev/pts", "devpts", MS_NOSUID | MS_NOEXEC, "newinstance,ptmxmode=0666,mode=0620");
	setupMount(ctx, "shm", "/dev/shm", "tmpfs", MS_NOEXEC | MS_NOSUID | MS_NODEV, "mode=1777,size=65536k");
	setupMount(ctx, "mqueue", "/dev/mqueue", "mqueue", MS_NOEXEC | MS_NOSUID | MS_NODEV, NULL);
	setupMount(ctx, "cgroup", "/sys/fs/cgroup", "cgroup2", MS_NOEXEC | MS_NOSUID | MS_NODEV | MS_RELATIME | MS_RDONLY, NULL);
	// Setup dev mounts.
	setupDevMount(ctx, "/dev/null", "/dev/null");
	setupDevMount(ctx, "/dev/random", "/dev/random");
	setupDevMount(ctx, "/dev/urandom", "/dev/urandom");
	// Pivot root.
	pivotRoot(ctx);
}

static inline void setupUtsNamespace(const Context* ctx) {
	ensure(sethostname("sandbox", strlen("sandbox")) == 0, "cannot set hostname");
}

static inline void prepareUserNamespace(int pid) {
	int fd;
	char path[64];
	char data[64];
	// Our process user has overflow UID and the same GID.
	// We can not directly change UID to 0 before making mapping.
	sprintf(path, "/proc/%d/uid_map", pid);
	sprintf(data, "%d %d %d\n", 0, getuid(), 1);
	fd = open(path, O_WRONLY | O_TRUNC);
	ensure(write(fd, data, strlen(data)) != -1, "cannot write uid_map");
	close(fd);
	// Before making groups mapping we should write "deny" into
	// "/proc/$PID/setgroups".
	sprintf(path, "/proc/%d/setgroups", pid);
	sprintf(data, "deny\n");
	fd = open(path, O_WRONLY | O_TRUNC);
	ensure(write(fd, data, strlen(data)) != -1, "cannot write setgroups");
	close(fd);
	// Now we can easily make mapping for groups.
	sprintf(path, "/proc/%d/gid_map", pid);
	sprintf(data, "%d %d %d\n", 0, getgid(), 1);
	fd = open(path, O_WRONLY | O_TRUNC);
	ensure(write(fd, data, strlen(data)) != -1, "cannot write gid_map");
	close(fd);
}

static inline void prepareCgroupNamespace(const Context* ctx) {
	if (rmdir(ctx->cgroupPath) != 0) {
		ensure(errno == ENOENT, "cannot remove cgroup");
	}
	if (mkdir(ctx->cgroupPath, 0755) != 0) {
		ensure(errno == EEXIST, "cannot create cgroup");
	}
	char* cgroupPath = malloc((strlen(ctx->cgroupPath) + strlen(CGROUP_MEMORY_SWAP_MAX_FILE) + 2) * sizeof(char));
	ensure(cgroupPath != 0, "cannot allocate cgroup path");
	// Limit max memory usage.
	{
		strcpy(cgroupPath, ctx->cgroupPath);
		strcat(cgroupPath, "/");
		strcat(cgroupPath, CGROUP_MEMORY_MAX_FILE);
		int fd = open(cgroupPath, O_WRONLY);
		ensure(fd != -1, "cannot open " CGROUP_MEMORY_MAX_FILE);
		char memoryStr[21];
		sprintf(memoryStr, "%d", ctx->memoryLimit);
		ensure(write(fd, memoryStr, strlen(memoryStr)) != -1, "cannot write " CGROUP_MEMORY_MAX_FILE);
		close(fd);
	}
	// Disable swap memory usage.
	{
		strcpy(cgroupPath, ctx->cgroupPath);
		strcat(cgroupPath, "/");
		strcat(cgroupPath, CGROUP_MEMORY_SWAP_MAX_FILE);
		int fd = open(cgroupPath, O_WRONLY);
		ensure(fd != -1, "cannot open " CGROUP_MEMORY_SWAP_MAX_FILE);
		ensure(write(fd, "0", strlen("0")) != -1, "cannot write " CGROUP_MEMORY_SWAP_MAX_FILE);
		close(fd);
	}
	// Limit process amount.
	// TODO: Replace hardcode with configurable limit
	{
		strcpy(cgroupPath, ctx->cgroupPath);
		strcat(cgroupPath, "/");
		strcat(cgroupPath, CGROUP_PIDS_MAX_FILE);
		int fd = open(cgroupPath, O_WRONLY);
		ensure(fd != -1, "cannot open " CGROUP_PIDS_MAX_FILE);
		char pidsStr[21];
		sprintf(pidsStr, "%d", ctx->pidsLimit);
		ensure(write(fd, pidsStr, strlen(pidsStr)) != -1, "cannot write " CGROUP_PIDS_MAX_FILE);
		close(fd);
	}
	// Limit CPU usage.
	if (ctx->flags & CPU_LIMIT_FLAG) {
		strcpy(cgroupPath, ctx->cgroupPath);
		strcat(cgroupPath, "/");
		strcat(cgroupPath, CGROUP_CPU_MAX_FILE);
		int fd = open(cgroupPath, O_WRONLY);
		ensure(fd != -1, "cannot open " CGROUP_CPU_MAX_FILE);
		char cpuStr[21];
		sprintf(cpuStr, "%d 100000", ctx->cpuLimit * 1000);
		ensure(write(fd, cpuStr, strlen(cpuStr)) != -1, "cannot write " CGROUP_CPU_MAX_FILE);
		close(fd);
	}
	free(cgroupPath);
}

static inline void initContext(Context* ctx, int argc, char* argv[]) {
	for (int i = 1; i < argc; ++i) {
		if (strcmp(argv[i], "--rootfs") == 0) {
			++i;
			ensure(i < argc, "--rootfs requires argument");
		} else if (strcmp(argv[i], "--overlay-upperdir") == 0) {
			++i;
			ensure(i < argc, "--overlay-upperdir requires argument");
		} else if (strcmp(argv[i], "--overlay-lowerdir") == 0) {
			++i;
			ensure(i < argc, "--overlay-lowerdir requires argument");
		} else if (strcmp(argv[i], "--overlay-workdir") == 0) {
			++i;
			ensure(i < argc, "--overlay-workdir requires argument");
		} else if (strcmp(argv[i], "--workdir") == 0) {
			++i;
			ensure(i < argc, "--workdir requires argument");
		} else if (strcmp(argv[i], "--env") == 0) {
			++i;
			ensure(i < argc, "--env requires argument");
			++ctx->environLen;
		} else if (strcmp(argv[i], "--cgroup-path") == 0) {
			++i;
			ensure(i < argc, "--cgroup-path requires argument");
		} else if (strcmp(argv[i], "--time-limit") == 0) {
			++i;
			ensure(i < argc, "--time-limit requires argument");
		} else if (strcmp(argv[i], "--memory-limit") == 0) {
			++i;
			ensure(i < argc, "--memory-limit requires argument");
		} else if (strcmp(argv[i], "--cpu-limit") == 0) {
			++i;
			ensure(i < argc, "--cpu-limit requires argument");
		} else if (strcmp(argv[i], "--pids-limit") == 0) {
			++i;
			ensure(i < argc, "--pids-limit requires argument");
		} else if (strcmp(argv[i], "--flags") == 0) {
			++i;
			ensure(i < argc, "--flags requires argument");
		} else if (strcmp(argv[i], "--report") == 0) {
			++i;
			ensure(i < argc, "--report requires argument");
		} else {
			ctx->argsLen = argc - i;
			break;
		}
	}
	ctx->args = malloc((ctx->argsLen + 1) * sizeof(char*));
	ensure(ctx->args != NULL, "cannot malloc arguments");
	ctx->args[ctx->argsLen] = NULL;
	ctx->environ = malloc((ctx->environLen + 1) * sizeof(char*));
	ensure(ctx->environ != NULL, "cannot malloc environ");
	ctx->environ[ctx->environLen] = NULL;
	int environIt = 0;
	for (int i = 1; i < argc; ++i) {
		if (strcmp(argv[i], "--rootfs") == 0) {
			++i;
			ctx->rootfs = argv[i];
		} else if (strcmp(argv[i], "--overlay-upperdir") == 0) {
			++i;
			ctx->overlayUpperdir = argv[i];
		} else if (strcmp(argv[i], "--overlay-lowerdir") == 0) {
			++i;
			ctx->overlayLowerdir = argv[i];
		} else if (strcmp(argv[i], "--overlay-workdir") == 0) {
			++i;
			ctx->overlayWorkdir = argv[i];
		} else if (strcmp(argv[i], "--workdir") == 0) {
			++i;
			ctx->workdir = argv[i];
		} else if (strcmp(argv[i], "--env") == 0) {
			++i;
			ctx->environ[environIt] = argv[i];
			++environIt;
		} else if (strcmp(argv[i], "--cgroup-path") == 0) {
			++i;
			ctx->cgroupPath = argv[i];
		} else if (strcmp(argv[i], "--time-limit") == 0) {
			++i;
			ensure(sscanf(argv[i], "%d", &ctx->timeLimit) == 1, "--time-limit has invalid argument");
		} else if (strcmp(argv[i], "--memory-limit") == 0) {
			++i;
			ensure(sscanf(argv[i], "%d", &ctx->memoryLimit) == 1, "--memory-limit has invalid argument");
		} else if (strcmp(argv[i], "--cpu-limit") == 0) {
			++i;
			ensure(sscanf(argv[i], "%d", &ctx->cpuLimit) == 1, "--cpu-limit has invalid argument");
		} else if (strcmp(argv[i], "--pids-limit") == 0) {
			++i;
			ensure(sscanf(argv[i], "%d", &ctx->pidsLimit) == 1, "--pids-limit has invalid argument");
		} else if (strcmp(argv[i], "--flags") == 0) {
			++i;
			ensure(sscanf(argv[i], "%d", &ctx->flags) == 1, "--flags has invalid argument");
		} else if (strcmp(argv[i], "--report") == 0) {
			++i;
			ctx->report = argv[i];
		} else {
			int argIt = 0;
			for (; i < argc; ++i) {
				ctx->args[argIt] = argv[i];
				++argIt;
			}
			ensure(argIt == ctx->argsLen, "corrupted argument count");
		}
	}
	ensure(environIt == ctx->environLen, "corrupted environ count");
}

static inline void waitReady(const Context* ctx) {
	char c;
	ensure(read(ctx->finalizePipe[0], &c, 1) == 0, "cannot wait finalize pipe to close");
	close(ctx->finalizePipe[0]);
}

static inline long getTimeDiff(struct timespec end, struct timespec begin) {
	return (long)(end.tv_sec - begin.tv_sec) * 1000 + (end.tv_nsec - begin.tv_nsec) / 1000000;
}

static inline Context* newContext() {
	Context* ctx = malloc(sizeof(Context));
	ensure(ctx != 0, "cannot allocate context");
	ctx->rootfs = "";
	ctx->overlayLowerdir = "";
	ctx->overlayUpperdir = "";
	ctx->overlayWorkdir = "";
	ctx->workdir = "/";
	ctx->args = NULL;
	ctx->argsLen = 0;
	ctx->environ = NULL;
	ctx->environLen = 0;
	ctx->cgroupPath = "";
	ctx->memoryLimit = 0;
	ctx->timeLimit = 0;
	ctx->cpuLimit = 0;
	ctx->pidsLimit = 32;
	ctx->flags = 0;
	ctx->report = "";
	// Uninitialized:
	//   ctx->initializePipe
	//   ctx->finalizePipe
	return ctx;
}

static inline void freeContext(Context* ctx) {
	free(ctx->args);
	free(ctx->environ);
	free(ctx);
}

int entrypoint(const Context* ctx) {
	close(ctx->initializePipe[1]);
	close(ctx->finalizePipe[0]);
	// Setup user namespace first of all.
	setupUserNamespace(ctx);
	setupMountNamespace(ctx);
	setupUtsNamespace(ctx);
	ensure(chdir(ctx->workdir) == 0, "cannot chdir to workdir");
	// Setup stack limit.
	struct rlimit limit;
	limit.rlim_cur = RLIM_INFINITY;
	limit.rlim_max = RLIM_INFINITY;
	ensure(setrlimit(RLIMIT_STACK, &limit) == 0, "cannot set stack limit");
	// Unlock parent process.
	close(ctx->finalizePipe[1]);
	return execvpe(ctx->args[0], ctx->args, ctx->environ);
}

static inline void readCgroupMemory(const char* path, long* value) {
	char data[21];
	int fd = open(path, O_RDONLY);
	ensure(fd != -1, "cannot open memory.current file");
	int bytes = read(fd, data, 20);
	ensure(bytes != -1 && bytes != EOF, "cannot read cgroup file");
	ensure(bytes > 0 && bytes <= 20, "invalid cgroup file size");
	data[bytes] = 0;
	*value = strtol(data, NULL, 10);
	ensure(*value != LONG_MAX, "invalid memory.current value");
	close(fd);
}

static inline void readCgroupCpuUsage(const char* path, long* value) {
	FILE* file = fopen(path, "re");
	ensure(file != NULL, "cannot open cpu.stat file");
	char* data = NULL;
	size_t len = 0;
	ssize_t bytes = 0;
	while ((bytes = getline(&data, &len, file)) != -1) {
		if (bytes <= 11 || memcmp(data, "usage_usec ", 11)) {
			continue;
		}
		*value = strtol(&data[11], NULL, 10);
		ensure(*value != LONG_MAX, "invalid cpu.stat usage_usec value");
	}
	fclose(file);
	free(data);
}

static inline void readCgroupOomCount(const Context* ctx, long* value) {
	char* filePath = malloc(strlen(ctx->cgroupPath) + strlen(CGROUP_MEMORY_EVENTS_FILE) + 2);
	ensure(filePath != NULL, "cannot allocate memory.events path");
	strcpy(filePath, ctx->cgroupPath);
	strcat(filePath, "/");
	strcat(filePath, CGROUP_MEMORY_EVENTS_FILE);
	FILE* file = fopen(filePath, "re");
	ensure(file != NULL, "cannot open memory.events file");
	char* data = NULL;
	size_t len = 0;
	ssize_t bytes = 0;
	while ((bytes = getline(&data, &len, file)) != -1) {
		if (bytes < 6 || memcmp(data, "oom ", 4)) {
			continue;
		}
		*value = strtol(&data[4], NULL, 10);
		ensure(*value != LONG_MAX, "invalid memory.events oom value");
	}
	fclose(file);
	free(data);
	free(filePath);
}

static volatile int cancelled = 0;

void cancel(int signal) {
	ensure(signal == SIGTERM, "received invalid signal");
	cancelled = 1;
}

int main(int argc, char* argv[]) {
	signal(SIGTERM, cancel);
	Context* ctx = newContext();
	initContext(ctx, argc, argv);
	ensure(ctx->argsLen, "empty execve arguments");
	ensure(strlen(ctx->rootfs), "--rootfs argument is required");
	ensure(strlen(ctx->overlayLowerdir), "--overlay-lowerdir is required");
	ensure(strlen(ctx->overlayUpperdir), "--overlay-upperdir is required");
	ensure(strlen(ctx->overlayWorkdir), "--overlay-workdir is required");
	ensure(strlen(ctx->cgroupPath), "--cgroup-path is required");
	ensure(ctx->timeLimit > 0, "--time-limit is required");
	ensure(ctx->memoryLimit > 0, "--memory-limit is required");
	ensure(!(ctx->flags & CPU_LIMIT_FLAG) || ctx->cpuLimit > 0, "--cpu-limit is required");
	ensure(pipe(ctx->initializePipe) == 0, "cannot create initialize pipe");
	ensure(pipe(ctx->finalizePipe) == 0, "cannot create finalize pipe");
	prepareCgroupNamespace(ctx);
	int cgroupFd = open(ctx->cgroupPath, O_PATH);
	ensure(cgroupFd != -1, "cannot open cgroup");
	struct clone_args cloneArgs = {};
	cloneArgs.flags = CLONE_NEWUSER | CLONE_NEWPID | CLONE_NEWNS | CLONE_NEWNET | CLONE_NEWIPC | CLONE_NEWUTS | CLONE_NEWCGROUP | CLONE_INTO_CGROUP;
	cloneArgs.cgroup = cgroupFd;
	pid_t pid = syscall(SYS_clone3, &cloneArgs, sizeof(cloneArgs));
	ensure(pid != -1, "cannot clone()");
	ensure(close(cgroupFd) == 0, "cannot close cgroup");
	if (pid == 0) {
		return entrypoint(ctx);
	}
	close(ctx->initializePipe[0]);
	close(ctx->finalizePipe[1]);
	// Setup user namespace.
	prepareUserNamespace(pid);
	// Setup cgroup file paths.
	char* memoryUsagePath = malloc(strlen(ctx->cgroupPath) + strlen(CGROUP_MEMORY_CURRENT_FILE) + 2);
	ensure(memoryUsagePath != NULL, "cannot allocate memory.current path");
	strcpy(memoryUsagePath, ctx->cgroupPath);
	strcat(memoryUsagePath, "/");
	if (!(ctx->flags & MEMORY_PEAK_FLAG)) {
		strcat(memoryUsagePath, CGROUP_MEMORY_CURRENT_FILE);
	} else {
		strcat(memoryUsagePath, CGROUP_MEMORY_PEAK_FILE);
	}
	char* cpuStatPath = malloc(strlen(ctx->cgroupPath) + strlen(CGROUP_CPU_STAT_FILE) + 2);
	ensure(cpuStatPath != NULL, "cannot allocate cpu.stat path");
	strcpy(cpuStatPath, ctx->cgroupPath);
	strcat(cpuStatPath, "/");
	strcat(cpuStatPath, CGROUP_CPU_STAT_FILE);
	// Now we should unlock child process.
	close(ctx->initializePipe[1]);
	//
	waitReady(ctx);
	struct timespec startTime, currentTime;
	ensure(clock_gettime(CLOCK_MONOTONIC, &startTime) == 0, "cannot get start time");
	int status;
	pid_t result;
	long memory = 0;
	long time = 0;
	long currentMemory = 0;
	struct timespec sleepSpec;
	sleepSpec.tv_sec = 0;
	sleepSpec.tv_nsec = 5000000;
	int realTimeLimit = ctx->timeLimit * 2;
	for (;;) {
		result = waitpid(pid, &status, WUNTRACED | WNOHANG | __WALL);
		if (result != 0) {
			break;
		}
		if (cancelled) {
			if (kill(pid, SIGKILL) != 0) {
				ensure(errno == ESRCH, "cannot kill process");
			}
			result = waitpid(pid, &status, WUNTRACED | __WALL);
			break;
		}
		ensure(clock_gettime(CLOCK_MONOTONIC, &currentTime) == 0, "cannot get current time");
		if (getTimeDiff(currentTime, startTime) > realTimeLimit) {
			if (kill(pid, SIGKILL) != 0) {
				ensure(errno == ESRCH, "cannot kill process");
			}
			result = waitpid(pid, &status, WUNTRACED | __WALL);
			break;
		}
		if (!(ctx->flags & MEMORY_PEAK_FLAG)) {
			readCgroupMemory(memoryUsagePath, &currentMemory);
			if (currentMemory > memory) {
				memory = currentMemory;
				if (memory > ctx->memoryLimit) {
					if (kill(pid, SIGKILL) != 0) {
						ensure(errno == ESRCH, "cannot kill process");
					}
					result = waitpid(pid, &status, WUNTRACED | __WALL);
					break;
				}
			}
		}
		readCgroupCpuUsage(cpuStatPath, &time);
		if (time > ctx->timeLimit * (long)1000) {
			if (kill(pid, SIGKILL) != 0) {
				ensure(errno == ESRCH, "cannot kill process");
			}
			result = waitpid(pid, &status, WUNTRACED | __WALL);
			break;
		}
		nanosleep(&sleepSpec, NULL);
	}
	ensure(result > 0, "cannot wait for child process");
	ensure(clock_gettime(CLOCK_MONOTONIC, &currentTime) == 0, "cannot get current time");
	readCgroupMemory(memoryUsagePath, &currentMemory);
	if (currentMemory > memory) {
		memory = currentMemory;
	}
	readCgroupCpuUsage(cpuStatPath, &time);
	int exitCode = WIFEXITED(status) ? WEXITSTATUS(status) : -1;
	if (exitCode != 0) {
		long oomCount = 0;
		readCgroupOomCount(ctx, &oomCount);
		if (oomCount > 0) {
			memory = ctx->memoryLimit + 1024;
		}
	}
	time /= 1000;
	long realTime = getTimeDiff(currentTime, startTime);
	if (time > ctx->timeLimit || realTime > realTimeLimit) {
		time = ctx->timeLimit + 1;
		realTime = realTimeLimit + 1;
	}
	if (strlen(ctx->report) != 0) {
		char line[60];
		int fd = open(ctx->report, O_WRONLY | O_TRUNC | O_CREAT, 0644);
		ensure(fd != -1, "cannot open report file");
		sprintf(line, "exit_code %d\n", exitCode);
		ensure(write(fd, line, strlen(line)) != -1, "cannot write report file");
		sprintf(line, "time %ld\n", time);
		ensure(write(fd, line, strlen(line)) != -1, "cannot write report file");
		sprintf(line, "real_time %ld\n", realTime);
		ensure(write(fd, line, strlen(line)) != -1, "cannot write report file");
		sprintf(line, "memory %ld\n", memory);
		ensure(write(fd, line, strlen(line)) != -1, "cannot write report file");
		close(fd);
	}
	free(cpuStatPath);
	free(memoryUsagePath);
	freeContext(ctx);
	return EXIT_SUCCESS;
}
