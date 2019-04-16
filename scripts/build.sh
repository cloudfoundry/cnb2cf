#!/usr/bin/env bash
set -euo pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )/.."

go build -o build/cnb2cf ./cmd/cnb2cf/main.go
