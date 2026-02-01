#!/bin/sh
set -xe

if [ ! -f cmd/finder.go ] ; then
    git clone https://github.com/LineageOS/android_build_soong.git --depth 1 --branch lineage-23.0
    cp android_build_soong/finder/{finder.go,fs,cmd} . -rf
    rm -rf android_build_soong
fi

go build -o finder ./cmd/
