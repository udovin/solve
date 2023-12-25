FROM golang:1.21-alpine3.18 AS build
RUN apk add --no-cache git gcc linux-headers libc-dev make
WORKDIR /src/solve
COPY go.mod go.sum /src/solve/
RUN go mod download -x
COPY . /src/solve
ARG VERSION=development
RUN make all

FROM alpine:3.18
RUN apk add --no-cache curl && \
    apk add --repository=https://dl-cdn.alpinelinux.org/alpine/edge/testing delve && \
    addgroup -S solve -g 1000 && adduser -S solve -G solve -u 1000
COPY --from=build /src/solve/cmd/solve/solve /src/solve/cmd/safeexec/safeexec /bin/
USER solve
VOLUME ["/tmp"]
ENV SOLVE_CONFIG=/etc/solve/config.json
ENTRYPOINT ["/bin/solve"]
