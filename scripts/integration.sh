#!/usr/bin/env bash
set -euo pipefail

function main() {
    pushd "$( dirname "${BASH_SOURCE[0]}" )/.." > /dev/null || return
        shim::bin::update
        ./scripts/build.sh

        go mod vendor

        set +e
            local exit_code
            go test -timeout 0 -mod=vendor ./integration/... -v -run Integration
            exit_code="${?}"

            if [[ "${exit_code}" != "0" ]]; then
                echo -e "\n\033[0;31m** GO Test Failed **\033[0m"
            else
                echo -e "\n\033[0;32m** GO Test Succeeded **\033[0m"
            fi
        set -e
    popd > /dev/null || return

    exit $exit_code
}

function shim::bin::update() {
    local cnb2cf_dir out_dir shim_dir
    cnb2cf_dir="$(realpath "$(dirname "${BASH_SOURCE[0]}")"/..)"
    out_dir="${cnb2cf_dir}/template/bin"
    shim_dir="${cnb2cf_dir}"

    pushd "${shim_dir}" > /dev/null || return
        GOOS=linux go build -ldflags="-s -w" -o "${out_dir}/detect" shims/cmd/detect/main.go
        GOOS=linux go build -ldflags="-s -w" -o "${out_dir}/supply" shims/cmd/supply/main.go
        GOOS=linux go build -ldflags="-s -w" -o "${out_dir}/finalize" shims/cmd/finalize/main.go
        GOOS=linux go build -ldflags="-s -w" -o "${out_dir}/release" shims/cmd/release/main.go
    popd > /dev/null || return
}

main
