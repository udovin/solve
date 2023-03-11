FROM alpine:3.17.0
ARG USER_ID=1000
ARG GROUP_ID=1000
RUN addgroup -g $GROUP_ID -S judge && \
    adduser -u $USER_ID -D -S -G judge -s /bin/sh judge && \
    apk add --no-cache \
    g++=12.2.1_git20220924-r4 && \
    rm -f /usr/libexec/gcc/x86_64-alpine-linux-musl/12.2.1/lto-wrapper \
    rm -f /usr/libexec/gcc/x86_64-alpine-linux-musl/12.2.1/cc1 \
    /usr/libexec/gcc/x86_64-alpine-linux-musl/12.2.1/lto1 \
    /usr/bin/lto-dump
USER judge:judge
WORKDIR /home/judge
