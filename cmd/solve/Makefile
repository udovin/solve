all: solve
.PHONY: solve
solve:
	go build -o solve -ldflags "-X github.com/udovin/solve/internal/config.Version=${VERSION}" .
clean:
	rm -f solve
