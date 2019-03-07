#!/usr/bin/env bash
set -exuo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

GOOS=linux go build -ldflags="-s -w" -o bin/detect github.com/cloudfoundry/libbuildpack/shims/cmd/detect
GOOS=linux go build -ldflags="-s -w" -o bin/supply github.com/cloudfoundry/libbuildpack/shims/cmd/supply
GOOS=linux go build -ldflags="-s -w" -o bin/finalize github.com/cloudfoundry/libbuildpack/shims/cmd/finalize
GOOS=linux go build -ldflags="-s -w" -o bin/release github.com/cloudfoundry/libbuildpack/shims/cmd/release
