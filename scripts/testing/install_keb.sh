#!/usr/bin/env bash
# Install KEB chart (handles both release and PR versions)
# Usage: install_keb.sh <is_release> <version>

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

IS_RELEASE=${1:?Is release flag required (true/false)}
VERSION=${2:?Version required}

if [ "$IS_RELEASE" == "true" ]; then
  make install VERSION="$VERSION"
else
  make install VERSION="PR-$VERSION" LOCAL_REGISTRY=true
fi

echo "KEB chart installed with version: $VERSION"
