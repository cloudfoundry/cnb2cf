#!/usr/bin/env bash
set -euo pipefail

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

function main() {
    pushd "${ROOT_DIR}" > /dev/null || return
        shim::bin::update

        cat <<COMPILE > ./template/bin/compile
#!/bin/bash
set -euo pipefail

BUILD_DIR=\$1
CACHE_DIR=\$2
export BUILDPACK_DIR=\`dirname \$(readlink -f \${BASH_SOURCE%/*})\`
export DEPS_DIR="\$BUILD_DIR/.cloudfoundry"
mkdir -p "\$DEPS_DIR/0"
mkdir -p "\$BUILD_DIR/.profile.d"
echo "export DEPS_DIR=\\\$HOME/.cloudfoundry" > "\$BUILD_DIR/.profile.d/0000_set-deps-dir.sh"

\$BUILDPACK_DIR/bin/supply "\$BUILD_DIR" "\$CACHE_DIR" "\$DEPS_DIR" 0
\$BUILDPACK_DIR/bin/finalize "\$BUILD_DIR" "\$CACHE_DIR" "\$DEPS_DIR" 0
COMPILE

        chmod +x ./template/bin/compile

        go get github.com/rakyll/statik

        export PATH="${PATH}:$(go env GOPATH)/bin"
        statik --src=./template -f --include '*'
        go build -o build/cnb2cf main.go
    popd > /dev/null || return
}

function shim::bin::update() {
    local out_dir
    out_dir="${ROOT_DIR}/template/bin"
    mkdir -p "${out_dir}"

    pushd "${ROOT_DIR}" > /dev/null || return
        for cmd in detect supply finalize release; do
            GOOS=linux go build -ldflags="-s -w" -o "${out_dir}/${cmd}" "shims/${cmd}/main.go"
        done
    popd > /dev/null || return
}

main
