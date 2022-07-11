FROM golang:1.18-alpine AS build
RUN apk add --no-cache git gcc linux-headers libc-dev
WORKDIR /src/solve
COPY go.mod go.sum /src/solve/
RUN go mod download -x
COPY . /src/solve
ARG VERSION=development
RUN go build -o solve -ldflags "-X github.com/udovin/solve/config.Version=${VERSION}" .

FROM alpine
RUN apk add --no-cache curl
RUN addgroup -S solve -g 1000 && adduser -S solve -G solve -u 1000
COPY --from=build /src/solve/solve /bin/solve
USER solve
VOLUME ["/tmp"]
ENTRYPOINT ["/bin/solve"]
