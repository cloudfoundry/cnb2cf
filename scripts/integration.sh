#!/usr/bin/env bash
set -euo pipefail

function main() {
    pushd "$( dirname "${BASH_SOURCE[0]}" )/.." > /dev/null || return
        ./scripts/build.sh

        set +e
            local exit_code
            go test -timeout 0 ./integration/... -v -run Integration
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

main
