#!/usr/bin/env bash
set -euo pipefail

function main() {
    pushd "$( dirname "${BASH_SOURCE[0]}" )/.." > /dev/null || return
        go get github.com/rakyll/statik
        statik -src=./template -f
        go build -o build/cnb2cf ./cmd/cnb2cf/main.go
    popd > /dev/null || return
}

main
