FROM golang:1.17-alpine AS build

RUN apk add --no-cache git gcc linux-headers libc-dev

COPY . /go/src/github.com/udovin/solve

WORKDIR /go/src/github.com/udovin/solve

RUN go get -d -v . && go build -o solve .

FROM alpine

COPY --from=build /go/src/github.com/udovin/solve/solve /bin/solve

RUN addgroup -S solve -g 1000 && adduser -S solve -G solve -u 1000

USER solve

ENTRYPOINT ["/bin/solve"]
