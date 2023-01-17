#define _GNU_SOURCE
#include <sched.h>
#include <signal.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <time.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <sys/sendfile.h>
#include <sys/stat.h>
#include <sys/syscall.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>

#define STACK_SIZE 16384
#define OVERLAY_DATA "lowerdir=%s,upperdir=%s,workdir=%s"
#define PROC_PATH "/proc"
#define CGROUP_PROCS_FILE "cgroup.procs"
#define CGROUP_MEMORY_MAX_FILE "memory.max"
#define CGROUP_MEMORY_SWAP_MAX_FILE "memory.swap.max"
#define OVERLAY_WORK ".work"

typedef struct {
	char* source;
	char* target;
} Mount;

typedef struct {
	int stdinFd;
	int stdoutFd;
	int stderrFd;
	char* rootfs;
	char* overlayLowerdir;
	char* overlayUpperdir;
	char* overlayWorkdir;
	char* workdir;
	char** args;
	int argsLen;
	Mount* inputFiles;
	int inputFilesLen;
	Mount* outputFiles;
	int outputFilesLen;
	char* cgroupPath;
	int memoryLimit;
	int timeLimit;
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

static inline void setupOverlayfs(Context* ctx) {
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

static inline void setupMount(Context* ctx, const char* source, const char* target, const char* device, unsigned long flags, const void* data) {
	char* path = malloc((strlen(ctx->rootfs) + strlen(target) + 1) * sizeof(char));
	ensure(path != 0, "cannot allocate");
	strcpy(path, ctx->rootfs);
	strcat(path, target);
	mkdirAll(strlen(ctx->rootfs), path);
	ensure(mount(source, path, device, flags, data) == 0, "cannot mount");
	free(path);
}

static inline void pivotRoot(Context* ctx) {
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

static inline void setupUserNamespace(Context* ctx) {
	// We should wait for setup of user namespace from parent.
	char c;
	ensure(read(ctx->initializePipe[0], &c, 1) == 0, "cannot wait initialize pipe to close");
	close(ctx->initializePipe[0]);
}

static inline void setupCgroupNamespace(Context* ctx) {
	ensure(unshare(CLONE_NEWCGROUP) == 0, "cannot unshare cgroup namespace");
}

static inline void setupMountNamespace(Context* ctx) {
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
	pivotRoot(ctx);
}

static inline void setupUtsNamespace(Context* ctx) {
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

static inline void prepareCgroupNamespace(Context* ctx, int pid) {
	if (rmdir(ctx->cgroupPath) != 0) {
		ensure(errno == ENOENT, "cannot remove cgroup");
	}
	if (mkdir(ctx->cgroupPath, 0755) != 0) {
		ensure(errno == EEXIST, "cannot create cgroup");
	}
	char* cgroupPath = malloc((strlen(ctx->cgroupPath) + strlen(CGROUP_MEMORY_SWAP_MAX_FILE) + 3) * sizeof(char));
	ensure(cgroupPath != 0, "cannot allocate cgroup path");
	{
		strcpy(cgroupPath, ctx->cgroupPath);
		strcat(cgroupPath, "/");
		strcat(cgroupPath, CGROUP_PROCS_FILE);
		int fd = open(cgroupPath, O_WRONLY);
		ensure(fd != -1, "cannot open cgroup.procs");
		char pidStr[21];
		sprintf(pidStr, "%d", pid);
		ensure(write(fd, pidStr, strlen(pidStr)) != -1, "cannot write cgroup.procs");
		close(fd);
	}
	{
		strcpy(cgroupPath, ctx->cgroupPath);
		strcat(cgroupPath, "/");
		strcat(cgroupPath, CGROUP_MEMORY_MAX_FILE);
		int fd = open(cgroupPath, O_WRONLY);
		ensure(fd != -1, "cannot open memory.max");
		char memoryStr[21];
		sprintf(memoryStr, "%d", ctx->memoryLimit);
		ensure(write(fd, memoryStr, strlen(memoryStr)) != -1, "cannot write cgroup.procs");
		close(fd);
	}
	{
		strcpy(cgroupPath, ctx->cgroupPath);
		strcat(cgroupPath, "/");
		strcat(cgroupPath, CGROUP_MEMORY_SWAP_MAX_FILE);
		int fd = open(cgroupPath, O_WRONLY);
		ensure(fd != -1, "cannot open memory.swap.max");
		char memoryStr[21];
		ensure(write(fd, "0", strlen("0")) != -1, "cannot write cgroup.procs");
		close(fd);
	}
	free(cgroupPath);
}

static inline void initContext(Context* ctx, int argc, char* argv[]) {
	for (int i = 1; i < argc; ++i) {
		if (strcmp(argv[i], "--stdin") == 0) {
			++i;
			ensure(i < argc, "--stdin requires argument");
		} else if (strcmp(argv[i], "--stdout") == 0) {
			++i;
			ensure(i < argc, "--stdout requires argument");
		} else if (strcmp(argv[i], "--stderr") == 0) {
			++i;
			ensure(i < argc, "--stderr requires argument");
		} else if (strcmp(argv[i], "--rootfs") == 0) {
			++i;
			ensure(i < argc, "--rootfs requires argument");
		} else if (strcmp(argv[i], "--overlay-upperdir") == 0) {
			++i;
			ensure(i < argc, "--overlay-upperdir requires argument");
		} else if (strcmp(argv[i], "--overlay-lowerdir") == 0) {
			++i;
			ensure(++i < argc, "--overlay-lowerdir requires argument");
		} else if (strcmp(argv[i], "--overlay-workdir") == 0) {
			++i;
			ensure(++i < argc, "--overlay-workdir requires argument");
		} else if (strcmp(argv[i], "--workdir") == 0) {
			++i;
			ensure(i < argc, "--workdir requires argument");
		} else if (strcmp(argv[i], "--input-file") == 0) {
			++i;
			ensure(i < argc, "--input-file requires two arguments");
			++i;
			ensure(i < argc, "--input-file requires two arguments");
			++ctx->inputFilesLen;
		} else if (strcmp(argv[i], "--output-file") == 0) {
			++i;
			ensure(i < argc, "--output-file requires two arguments");
			++i;
			ensure(i < argc, "--output-file requires two arguments");
			++ctx->outputFilesLen;
		} else if (strcmp(argv[i], "--cgroup-path") == 0) {
			++i;
			ensure(i < argc, "--cgroup-path requires argument");
		} else if (strcmp(argv[i], "--time-limit") == 0) {
			++i;
			ensure(i < argc, "--time-limit requires argument");
		} else if (strcmp(argv[i], "--memory-limit") == 0) {
			++i;
			ensure(i < argc, "--memory-limit requires argument");
		} else if (strcmp(argv[i], "--report") == 0) {
			++i;
			ensure(i < argc, "--report requires argument");
		} else {
			ctx->argsLen = argc - i;
			break;
		}
	}
	ctx->args = malloc((ctx->argsLen + 1) * sizeof(char*));
	ensure(ctx->args != 0, "cannot malloc arguments");
	ctx->args[ctx->argsLen] = 0;
	if (ctx->inputFilesLen) {
		ctx->inputFiles = malloc(ctx->inputFilesLen * sizeof(Mount));
		ensure(ctx->inputFiles != 0, "cannot malloc input files");
	}
	if (ctx->outputFilesLen) {
		ctx->outputFiles = malloc(ctx->outputFilesLen * sizeof(Mount));
		ensure(ctx->outputFiles != 0, "cannot malloc output files");
	}
	int inputFileIt = 0;
	int outputFileIt = 0;
	for (int i = 1; i < argc; ++i) {
		if (strcmp(argv[i], "--stdin") == 0) {
			++i;
			ctx->stdinFd = open(argv[i], O_RDONLY);
			ensure(ctx->stdinFd != -1, "cannot open stdin file");
		} else if (strcmp(argv[i], "--stdout") == 0) {
			++i;
			ctx->stdoutFd = open(argv[i], O_WRONLY | O_TRUNC | O_CREAT, 0644);
			ensure(ctx->stdoutFd != -1, "cannot open stdout file");
		} else if (strcmp(argv[i], "--stderr") == 0) {
			++i;
			ctx->stderrFd = open(argv[i], O_WRONLY | O_TRUNC | O_CREAT, 0644);
			ensure(ctx->stderrFd != -1, "cannot open stderr file");
		} else if (strcmp(argv[i], "--rootfs") == 0) {
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
		} else if (strcmp(argv[i], "--input-file") == 0) {
			++i;
			ctx->inputFiles[inputFileIt].source = argv[i];
			++i;
			ctx->inputFiles[inputFileIt].target = argv[i];
			++inputFileIt;
		} else if (strcmp(argv[i], "--output-file") == 0) {
			++i;
			ctx->outputFiles[outputFileIt].source = argv[i];
			++i;
			ctx->outputFiles[outputFileIt].target = argv[i];
			++outputFileIt;
		} else if (strcmp(argv[i], "--cgroup-path") == 0) {
			++i;
			ctx->cgroupPath = argv[i];
		} else if (strcmp(argv[i], "--time-limit") == 0) {
			++i;
			ensure(sscanf(argv[i], "%d", &ctx->timeLimit) == 1, "--time-limit has invalid argument");
		} else if (strcmp(argv[i], "--memory-limit") == 0) {
			++i;
			ensure(sscanf(argv[i], "%d", &ctx->memoryLimit) == 1, "--memory-limit has invalid argument");
		} else if (strcmp(argv[i], "--report") == 0) {
			++i;
			ctx->report = argv[i];
		} else {
			for (int j = 0; i < argc; ++j) {
				ctx->args[j] = argv[i];
				++i;
			}
		}
	}
}

static inline void copyFile(char* target, char* source) {
	int input = open(source, O_RDONLY);
	ensure(input != -1, "cannot open source file");
	struct stat fileinfo = {};
	ensure(fstat(input, &fileinfo) == 0, "cannot fstat source file");
	int output = open(source, O_WRONLY | O_TRUNC | O_CREAT, fileinfo.st_mode);
	ensure(output != -1, "cannot open target file");
	off_t bytesCopied = 0;
	ensure(sendfile(output, input, &bytesCopied, fileinfo.st_size) != -1, "cannot copy source to target");
}

static inline void waitReady(Context* ctx) {
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
	ctx->stdinFd = -1;
	ctx->stdoutFd = -1;
	ctx->stderrFd = -1;
	ctx->rootfs = "";
	ctx->overlayLowerdir = "";
	ctx->overlayUpperdir = "";
	ctx->overlayWorkdir = "";
	ctx->workdir = "/";
	ctx->args = NULL;
	ctx->argsLen = 0;
	ctx->inputFiles = NULL;
	ctx->inputFilesLen = 0;
	ctx->outputFiles = NULL;
	ctx->outputFilesLen = 0;
	ctx->cgroupPath = "";
	ctx->timeLimit = 0;
	ctx->memoryLimit = 0;
	ctx->report = "";
	return ctx;
}

static inline void freeContext(Context* ctx) {
	free(ctx->args);
	free(ctx->inputFiles);
	free(ctx->outputFiles);
	free(ctx);
}

int entrypoint(void* arg) {
	ensure(arg != 0, "cannot get config");
	Context* ctx = (Context*)arg;
	close(ctx->initializePipe[1]);
	close(ctx->finalizePipe[0]);
	// Setup user namespace first of all.
	setupUserNamespace(ctx);
	setupCgroupNamespace(ctx);
	setupMountNamespace(ctx);
	setupUtsNamespace(ctx);
	ensure(chdir(ctx->workdir) == 0, "cannot chdir to workdir");
	if (ctx->stdinFd != -1) {
		ensure(dup2(ctx->stdinFd, STDIN_FILENO) != -1, "cannot setup stdin");
		close(ctx->stdinFd);
	}
	if (ctx->stdoutFd != -1) {
		ensure(dup2(ctx->stdoutFd, STDOUT_FILENO) != -1, "cannot setup stdout");
		close(ctx->stdoutFd);
	}
	if (ctx->stderrFd != -1) {
		ensure(dup2(ctx->stderrFd, STDERR_FILENO) != -1, "cannot setup stderr");
		close(ctx->stderrFd);
	}
	close(ctx->finalizePipe[1]);
	execvpe(ctx->args[0], ctx->args, NULL);
}

int main(int argc, char* argv[]) {
	Context* ctx = newContext();
	initContext(ctx, argc, argv);
	ensure(ctx->argsLen, "empty execve arguments");
	ensure(strlen(ctx->rootfs), "--rootfs argument is required");
	ensure(strlen(ctx->overlayLowerdir), "--overlay-lowerdir is required");
	ensure(strlen(ctx->overlayUpperdir), "--overlay-upperdir is required");
	ensure(strlen(ctx->overlayWorkdir), "--overlay-workdir is required");
	ensure(strlen(ctx->cgroupPath), "--cgroup-path is required");
	ensure(ctx->timeLimit, "--time-limit is required");
	ensure(ctx->memoryLimit, "--memory-limit is required");
	ensure(pipe(ctx->initializePipe) == 0, "cannot create initialize pipe");
	ensure(pipe(ctx->finalizePipe) == 0, "cannot create finalize pipe");
	char* stack = malloc(STACK_SIZE);
	ensure(stack != 0, "cannot allocate stack");
	int pid = clone(
		entrypoint,
		stack + STACK_SIZE,
		CLONE_NEWUSER | CLONE_NEWPID | CLONE_NEWNS | CLONE_NEWNET | CLONE_NEWIPC | CLONE_NEWUTS,
		ctx);
	free(stack);
	ensure(pid != -1, "cannot clone()");
	close(ctx->initializePipe[0]);
	close(ctx->finalizePipe[1]);
	if (ctx->stdinFd != -1) { close(ctx->stdinFd); }
	if (ctx->stdoutFd != -1) { close(ctx->stdoutFd); }
	if (ctx->stderrFd != -1) { close(ctx->stderrFd); }
	// Setup user namespace.
	prepareUserNamespace(pid);
	// Setup cgroup namespace.
	prepareCgroupNamespace(ctx, pid);
	// Now we should unlock child process.
	close(ctx->initializePipe[1]);
	//
	waitReady(ctx);
	struct timespec startTime, currentTime;
	ensure(clock_gettime(CLOCK_MONOTONIC, &startTime) == 0, "cannot get start time");
	int status;
	pid_t result;
	do {
		result = waitpid(pid, &status, WUNTRACED | WNOHANG | __WALL);
		if (result < 0) {
			ensure(errno == EINTR, "cannot wait for child process");
		}
		ensure(clock_gettime(CLOCK_MONOTONIC, &currentTime) == 0, "cannot get current time");
		if (result == 0 && getTimeDiff(currentTime, startTime) > ctx->timeLimit) {
			if (kill(pid, SIGKILL) != 0) {
				ensure(errno == ESRCH, "cannot kill process");
			}
		}
		usleep(5000);
	} while (result == 0);
	if (strlen(ctx->report) != 0) {
		char line[60];
		int fd = open(ctx->report, O_WRONLY | O_TRUNC | O_CREAT, 0644);
		ensure(fd != -1, "cannot open report file");
		sprintf(line, "time %ld\n", getTimeDiff(currentTime, startTime));
		ensure(write(fd, line, strlen(line)) != -1, "cannot write report file");
		sprintf(line, "memory %d\n", 1024);
		ensure(write(fd, line, strlen(line)) != -1, "cannot write report file");
		sprintf(line, "exit_code %d\n", WIFEXITED(status) ? WEXITSTATUS(status) : -1);
		ensure(write(fd, line, strlen(line)) != -1, "cannot write report file");
		close(fd);
	}
	freeContext(ctx);
	return EXIT_SUCCESS;
}
