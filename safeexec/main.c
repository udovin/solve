#define _GNU_SOURCE
#include <sched.h>
#include <signal.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/mount.h>
#include <sys/wait.h>

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
    int stdin;
    int stdout;
    int stderr;
    char* rootfs;
    int argc;
    char** argv;
    int pipe[2];
} Config;

#define STACK_SIZE 8096

int entrypoint(void* arg) {
    ensure(arg != 0, "cannot get config");
    Config* config = (Config*)arg;
    close(config->pipe[1]);
	// We should wait for setup of user namespace from parent.
    char c;
	ensure(read(config->pipe[0], &c, 1) == 0, "cannot wait pipe to close");
    close(config->pipe[0]);
    printf("pid = %d\n", getpid());
    printf("uid = %d\n", getuid());
    printf("gid = %d\n", getgid());
    printf("stdin = %d\n", config->stdin);
    printf("stdout = %d\n", config->stdout);
    printf("stderr = %d\n", config->stderr);
    printf("rootfs = %s\n", config->rootfs);
    if (mount("/", "/", NULL, MS_REC | MS_PRIVATE, NULL) == -1) {
        return EXIT_FAILURE;
    }
    if (config->stdin != -1) {
        ensure(dup2(config->stdin, STDIN_FILENO) != -1, "cannot setup stdin");
        close(config->stdin);
    }
    if (config->stdout != -1) {
        ensure(dup2(config->stdout, STDOUT_FILENO) != -1, "cannot setup stdout");
        close(config->stdout);
    }
    if (config->stderr != -1) {
        ensure(dup2(config->stderr, STDERR_FILENO) != -1, "cannot setup stderr");
        close(config->stderr);
    }
    execve(config->argv[0], config->argv, 0);
}

int main(int argc, char* argv[]) {
    Config config;
    config.stdin = -1;
    config.stdout = -1;
    config.stderr = -1;
    config.rootfs = "";
    config.argc = 0;
    config.argv = 0;
    for (int i = 1; i < argc; ++i) {
        if (strcmp(argv[i], "--stdin") == 0) {
            ++i;
            ensure(i < argc, "--stdin requires argument");
            config.stdin = open(argv[i], O_RDONLY);
            ensure(config.stdin != -1, "cannot open stdin file");
        } else if (strcmp(argv[i], "--stdout") == 0) {
            ++i;
            ensure(i < argc, "--stdout requires argument");
            config.stdout = open(argv[i], O_WRONLY | O_TRUNC | O_CREAT, 0644);
            ensure(config.stdout != -1, "cannot open stdout file");
        } else if (strcmp(argv[i], "--stderr") == 0) {
            ++i;
            ensure(i < argc, "--stderr requires argument");
            config.stderr = open(argv[i], O_WRONLY | O_TRUNC | O_CREAT, 0644);
            ensure(config.stderr != -1, "cannot open stderr file");
        } else if (strcmp(argv[i], "--rootfs") == 0) {
            ++i;
            ensure(i < argc, "--rootfs requires argument");
            config.rootfs = argv[i];
        } else {
            config.argc = argc - i;
            config.argv = malloc((config.argc + 1) * sizeof(const char*));
            ensure(config.argv != 0, "cannot malloc argv");
            config.argv[config.argc] = 0;
            for (int j = 0; i < argc; ++j) {
                config.argv[j] = argv[i];
                ++i;
            }
        }
    }
    ensure(strlen(config.rootfs), "--rootfs argument is required");
    ensure(config.argv > 0, "at least one exec argument is required");
    ensure(pipe(config.pipe) != -1, "cannot create pipe");
    char* stack = malloc(STACK_SIZE);
    ensure(stack != 0, "cannot allocate stack");
    int pid = clone(
        entrypoint,
        stack + STACK_SIZE,
        SIGCHLD | CLONE_NEWUSER | CLONE_NEWPID | CLONE_NEWNS | CLONE_NEWNET | CLONE_NEWIPC | CLONE_NEWUTS | CLONE_NEWCGROUP,
        &config
    );
    ensure(pid != -1, "cannot clone()");
    close(config.pipe[0]);
    if (config.stdin != -1) {
        close(config.stdin);
    }
    if (config.stdout != -1) {
        close(config.stdout);
    }
    if (config.stderr != -1) {
        close(config.stderr);
    }
    //
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
    // Now we should unlock child process.
	close(config.pipe[1]);
    //
    int status;
    waitpid(pid, &status, 0);
    printf("exit code = %d\n", WEXITSTATUS(status));
    return EXIT_SUCCESS;
}
