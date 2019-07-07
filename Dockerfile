FROM golang:alpine AS build

RUN apk add --no-cache git gcc libc-dev

COPY . /go/src/github.com/udovin/solve

WORKDIR /go/src/github.com/udovin/solve

RUN go get -d -v . && go build -o solve .

FROM alpine

COPY --from=build /go/src/github.com/udovin/solve/solve /bin/solve

ENTRYPOINT ["/bin/solve"]
