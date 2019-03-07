#!/usr/bin/env bash
set -euo pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )/.."

./scripts/build.sh

go test ./... -run Integration
