FROM golang:1.18-alpine AS build

RUN apk add --no-cache git gcc linux-headers libc-dev

COPY . /go/src/github.com/udovin/solve

WORKDIR /go/src/github.com/udovin/solve

RUN go get -d -v . && go build -o solve .

FROM alpine

RUN apk add --no-cache curl

RUN addgroup -S solve -g 1000 && adduser -S solve -G solve -u 1000

COPY --from=build /go/src/github.com/udovin/solve/solve /bin/solve

USER solve

VOLUME ["/tmp"]

ENTRYPOINT ["/bin/solve"]
