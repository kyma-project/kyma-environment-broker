#!/usr/bin/env bash
# This script creates a folder with notable changes for given version
# It has the following arguments:
#   - version (mandatory)
# ./create_notable_change_dir.sh 1.22.0

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

SOURCE_DIR="notable-changes-to-release/"

if [ -z "$( ls -A ${SOURCE_DIR} )" ]; then
   echo "No notable changes to move"
   exit 0
fi

if [ -d "notable-changes/$1" ]; then
   echo "Notable changes for version $1 already exist"
   exit 1
fi

mkdir -p notable-changes/$1

mv notable-changes-to-release/* notable-changes/$1/