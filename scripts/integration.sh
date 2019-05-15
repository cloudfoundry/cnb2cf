#!/usr/bin/env bash
set -euo pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )/.."

./scripts/build.sh
go mod vendor

echo "Run CNB2CF Runtime Integration Tests"
set +e
go test -timeout 0 -mod=vendor ./integration/... -v -run IntegrationCrea
exit_code=$?

if [ "$exit_code" != "0" ]; then
    echo -e "\n\033[0;31m** GO Test Failed **\033[0m"
else
    echo -e "\n\033[0;32m** GO Test Succeeded **\033[0m"
fi

exit $exit_code
