#!/bin/bash -l

set -euo pipefail

cnb2cf="$(realpath $(dirname "${BASH_SOURCE[0]}")/..)"
out="$cnb2cf/template/bin/"
while getopts "s:" arg
do
    case $arg in
    s) shim="${OPTARG}";;
    *) echo "usage -s <PATH_TO_LIBBUILDPACK>" && return 1;
    esac
done

pushd "$shim"
    GOOS=linux go build -ldflags="-s -w" -o "$out/detect" shims/cmd/detect/main.go
    GOOS=linux go build -ldflags="-s -w" -o "$out/supply" shims/cmd/supply/main.go
    GOOS=linux go build -ldflags="-s -w" -o "$out/finalize" shims/cmd/finalize/main.go
    GOOS=linux go build -ldflags="-s -w" -o "$out/release" shims/cmd/release/main.go
popd

pushd "$cnb2cf"
  go get github.com/rakyll/statik
  statik -src=./template -f
popd
