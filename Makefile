all: solve safeexec
.PHONY: solve safeexec
VERSION ?= development
solve:
	go build -o solve -ldflags "-X github.com/udovin/solve/config.Version=${VERSION}" .
test: safeexec
	go test ./...
test-reset: safeexec
	TEST_RESET_DATA=1 go test ./...
clean:
	rm -f solve
	@$(MAKE) --no-print-directory -C safeexec clean
safeexec:
	@$(MAKE) --no-print-directory -C $@
