all: solve safeexec
.PHONY: solve safeexec
VERSION ?= development
solve:
	go build -o solve -ldflags "-X github.com/udovin/solve/config.Version=${VERSION}" .
test:
	go test ./...
test-reset:
	TEST_RESET_DATA=1 go test ./...
clean:
	rm -f solve
	$(MAKE) -C safeexec clean
safeexec:
	$(MAKE) -C $@
