#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Lux root directory
GETH_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )

# Load the versions
source "$GETH_PATH"/scripts/versions.sh

# Load the constants
source "$GETH_PATH"/scripts/constants.sh

echo "Building Docker Image: $dockerhub_repo:$build_image_id based of $lux_version"
docker build -t "$dockerhub_repo:$build_image_id" "$GETH_PATH" -f "$GETH_PATH/Dockerfile" \
  --build-arg LUX_VERSION="$lux_version" \
  --build-arg GETH_COMMIT="$geth_commit" \
  --build-arg CURRENT_BRANCH="$current_branch"
