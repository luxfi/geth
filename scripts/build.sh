#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Root directory
GETH_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )

# Load the versions
source "$GETH_PATH"/scripts/versions.sh

# Load the constants
source "$GETH_PATH"/scripts/constants.sh

if [[ $# -eq 1 ]]; then
    binary_path=$1
elif [[ $# -ne 0 ]]; then
    echo "Invalid arguments to build geth. Requires either no arguments (default) or one arguments to specify binary location."
    exit 1
fi

# Check if GETH_COMMIT is set, if not retrieve the last commit from the repo.
# This is used in the Dockerfile to allow a commit hash to be passed in without
# including the .git/ directory within the Docker image.
GETH_COMMIT=${GETH_COMMIT:-$(git rev-list -1 HEAD)}

# Build Geth, which runs as a subprocess
echo "Building Geth @ GitCommit: $GETH_COMMIT"
go build -ldflags "-X github.com/luxfi/geth/plugin/evm.GitCommit=$GETH_COMMIT" -o "$binary_path" "plugin/"*.go
