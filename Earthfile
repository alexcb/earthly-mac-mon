FROM golang:1.13-alpine3.11

RUN apk add --update --no-cache \
    bash \
    bash-completion \
    binutils \
    ca-certificates \
    coreutils \
    curl \
    findutils \
    g++ \
    git \
    grep \
    less \
    make \
    openssl \
    util-linux

WORKDIR /earthly

deps:
    RUN go get golang.org/x/tools/cmd/goimports
    RUN go get golang.org/x/lint/golint
    RUN go get github.com/gordonklaus/ineffassign
    COPY go.mod go.sum ./
    RUN go mod download
    SAVE ARTIFACT go.mod AS LOCAL go.mod
    SAVE ARTIFACT go.sum AS LOCAL go.sum
    SAVE IMAGE

code:
    FROM +deps
    COPY --dir cmd slack ./
    SAVE IMAGE


build:
    FROM +code
    ARG GOOS=darwin
    ARG GOARCH=amd64
    ARG GOCACHE=/go-cache
    RUN --mount=type=cache,target=$GOCACHE \
        go build \
            -o build/earth \
            cmd/earthlymacmon/*.go
    SAVE ARTIFACT ./build/tags
    SAVE ARTIFACT ./build/ldflags
    SAVE ARTIFACT build/earth AS LOCAL "build/$GOOS/$GOARCH/earth"
