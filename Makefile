all: solve safeexec
.PHONY: solve safeexec
solve:
	@$(MAKE) --no-print-directory -C cmd/solve
safeexec:
	@$(MAKE) --no-print-directory -C cmd/safeexec
compiler-test:
	@$(MAKE) --no-print-directory -C cmd/compiler-test
test: safeexec
	go test -race ./...
test-reset: safeexec
	TEST_RESET_DATA=1 go test ./...
clean:
	@$(MAKE) --no-print-directory -C cmd/solve clean
	@$(MAKE) --no-print-directory -C cmd/safeexec clean
	@$(MAKE) --no-print-directory -C cmd/compiler-test clean
