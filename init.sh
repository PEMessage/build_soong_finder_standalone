#!/bin/sh
set -xe

if [ ! -f cmd/finder.go ] ; then
    git clone https://github.com/LineageOS/android_build_soong.git --depth 1 --branch lineage-23.0
    cp android_build_soong/finder/{finder.go,fs,cmd} . -rf
    rm -rf android_build_soong
fi

go build -o finder ./cmd/

if command -v zig > /dev/null ; then
    # Thanks to: https://calabro.io/zig-cgo
    # #            https://github.com/golang/go/issues/56386#issuecomment-1289185008
    (
        export CC="zig cc -target x86_64-linux-musl"
        export CGO_ENABLED=1
        export CGO_LDFLAGS="-static"
        export GOOS=linux GOARCH=amd64
        go build -a -ldflags '-extldflags "-static"' -ldflags=-linkmode=external -o finder_static ./cmd/
        go build -a -ldflags '-extldflags "-static"' -ldflags=-linkmode=external -o readdb ./main/
    )
fi


