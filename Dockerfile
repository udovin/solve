FROM golang:alpine AS build

RUN apk add --no-cache git gcc libc-dev

COPY source /build

WORKDIR /build

RUN go get -d -v . && go build -o solve .

FROM alpine

COPY --from=build /build/solve /bin/solve

ENTRYPOINT ["/bin/solve"]
