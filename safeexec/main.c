#define _GNU_SOURCE
#include <sched.h>
#include <signal.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <sys/sendfile.h>
#include <sys/stat.h>
#include <sys/syscall.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>

void ensure(int value, const char* message) {
	if (!value) {
		puts(message);
		exit(EXIT_FAILURE);
	}
}

typedef struct {
	char* source;
	char* target;
} Mount;

typedef struct {
	int stdinFd;
	int stdoutFd;
	int stderrFd;
	char* rootfs;
	char* lowerdir;
	char* upperdir;
	char* workdir;
	char** args;
	int argsLen;
	Mount* inputFiles;
	int inputFilesLen;
	Mount* outputFiles;
	int outputFilesLen;
	int pipe[2];
	char* overlayWorkdir;
} Context;

#define STACK_SIZE 16384
#define OVERLAY_DATA "lowerdir=%s,upperdir=%s,workdir=%s"
#define PROC_PATH "/proc"

void setupUserNamespace(Context* ctx) {
	// We should wait for setup of user namespace from parent.
	char c;
	ensure(read(ctx->pipe[0], &c, 1) == 0, "cannot wait pipe to close");
	close(ctx->pipe[0]);
}

void setupOverlayfs(Context* ctx) {
	char* data = malloc((strlen(ctx->lowerdir) + strlen(ctx->upperdir) + strlen(ctx->overlayWorkdir) + strlen(OVERLAY_DATA)) * sizeof(char));
	ensure(data != 0, "cannot allocate rootfs overlay data");
	sprintf(data, OVERLAY_DATA, ctx->lowerdir, ctx->upperdir, ctx->overlayWorkdir);
	ensure(mount("overlay", ctx->rootfs, "overlay", 0, data) == 0, "cannot mount rootfs overlay");
}

void setupProcfs(Context* ctx) {
	char* path = malloc((strlen(ctx->rootfs) + strlen(PROC_PATH)) * sizeof(char));
	ensure(path != 0, "cannot allocate proc path");
	strcat(path, ctx->rootfs);
	strcat(path, PROC_PATH);
	ensure(mount("proc", path, "proc", MS_NOEXEC | MS_NOSUID | MS_NODEV, NULL) == 0, "cannot mount \"/proc\"");
}

void pivotRoot(Context* ctx) {
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

void setupMountNamespace(Context* ctx) {
	// First of all make all changes are private for current root.
	ensure(mount(NULL, "/", NULL, MS_SLAVE | MS_REC, NULL) == 0, "cannot remount \"/\"");
	ensure(mount(NULL, "/", NULL, MS_PRIVATE, NULL) == 0, "cannot remount \"/\"");
	ensure(mount(ctx->rootfs, ctx->rootfs, "bind", MS_BIND | MS_REC, NULL) == 0, "cannot remount rootfs");
	setupOverlayfs(ctx);
	setupProcfs(ctx);
	pivotRoot(ctx);
}

int entrypoint(void* arg) {
	ensure(arg != 0, "cannot get config");
	Context* ctx = (Context*)arg;
	close(ctx->pipe[1]);
	// Setup user namespace first of all.
	setupUserNamespace(ctx);
	setupMountNamespace(ctx);
	chdir(ctx->workdir);
	printf("pid = %d\n", getpid());
	printf("uid = %d\n", getuid());
	printf("gid = %d\n", getgid());
	printf("stdin = %d\n", ctx->stdinFd);
	printf("stdout = %d\n", ctx->stdoutFd);
	printf("stderr = %d\n", ctx->stderrFd);
	printf("rootfs = %s\n", ctx->rootfs);
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
	execve(ctx->args[0], ctx->args, 0);
}

void prepareUserNamespace(int pid) {
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

void initContext(Context* ctx, int argc, char* argv[]) {
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
		} else if (strcmp(argv[i], "--upperdir") == 0) {
			++i;
			ensure(i < argc, "--upperdir requires argument");
		} else if (strcmp(argv[i], "--lowerdir") == 0) {
			++i;
			ensure(i < argc, "--lowerdir requires argument");
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
		} else {
			ctx->argsLen = argc - i;
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
		} else if (strcmp(argv[i], "--upperdir") == 0) {
			++i;
			ctx->upperdir = argv[i];
		} else if (strcmp(argv[i], "--lowerdir") == 0) {
			++i;
			ctx->lowerdir = argv[i];
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
		} else {
			for (int j = 0; i < argc; ++j) {
				ctx->args[j] = argv[i];
				++i;
			}
		}
	}
}

void copyFile(char* target, char* source) {
	int input = open(source, O_RDONLY);
	ensure(input != -1, "cannot open source file");
	struct stat fileinfo = {};
	ensure(fstat(input, &fileinfo) == 0, "cannot fstat source file");
	int output = open(source, O_WRONLY | O_TRUNC | O_CREAT, fileinfo.st_mode);
	ensure(output != -1, "cannot open target file");
	off_t bytesCopied = 0;
	ensure(sendfile(output, input, &bytesCopied, fileinfo.st_size) != -1, "cannot copy source to target");
}

int main(int argc, char* argv[]) {
	Context* ctx = malloc(sizeof(Context));
	ensure(ctx != 0, "cannot allocate context");
	ctx->stdinFd = -1;
	ctx->stdoutFd = -1;
	ctx->stderrFd = -1;
	ctx->rootfs = "";
	ctx->lowerdir = "";
	ctx->upperdir = "";
	ctx->workdir = "/";
	ctx->args = 0;
	ctx->argsLen = 0;
	ctx->inputFiles = 0;
	ctx->inputFilesLen = 0;
	ctx->outputFiles = 0;
	ctx->outputFilesLen = 0;
	initContext(ctx, argc, argv);
	ensure(ctx->argsLen, "empty execve arguments");
	ensure(strlen(ctx->rootfs), "--rootfs argument is required");
	ensure(strlen(ctx->lowerdir), "--lowerdir is required");
	ensure(strlen(ctx->upperdir), "--upperdir is required");
	ensure(pipe(ctx->pipe) == 0, "cannot create pipe");
	ctx->overlayWorkdir = malloc((strlen(ctx->upperdir) + 5) * sizeof(char));
	ensure(ctx->overlayWorkdir != 0, "cannot allocate overlay workdir path");
	ctx->overlayWorkdir[0] = 0;
	strcat(ctx->overlayWorkdir, ctx->upperdir);
	strcat(ctx->overlayWorkdir, ".work");
	ensure(mkdir(ctx->overlayWorkdir, 0777) == 0, "cannot create overlay workdir");
	char* stack = malloc(STACK_SIZE);
	ensure(stack != 0, "cannot allocate stack");
	int pid = clone(
		entrypoint,
		stack + STACK_SIZE,
		SIGCHLD | CLONE_NEWUSER | CLONE_NEWPID | CLONE_NEWNS | CLONE_NEWNET | CLONE_NEWIPC | CLONE_NEWUTS | CLONE_NEWCGROUP,
		ctx);
	ensure(pid != -1, "cannot clone()");
	close(ctx->pipe[0]);
	if (ctx->stdinFd != -1) { close(ctx->stdinFd); }
	if (ctx->stdoutFd != -1) { close(ctx->stdoutFd); }
	if (ctx->stderrFd != -1) { close(ctx->stderrFd); }
	// Setup user namespace.
	prepareUserNamespace(pid);
	// Now we should unlock child process.
	close(ctx->pipe[1]);
	//
	int status;
	waitpid(pid, &status, 0);
	printf("exit code = %d\n", WEXITSTATUS(status));
	return EXIT_SUCCESS;
}
