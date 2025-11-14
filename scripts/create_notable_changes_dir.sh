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
TARGET_DIR="notable-changes/$1/"

# check if there is anything to move
if [ -z "$( find ${SOURCE_DIR} -type f -not -name README.md )" ]; then
   echo "No notable changes to move"
   exit 0
fi

echo "Creating notable changes"

# create target directory
mkdir -p ${TARGET_DIR}

# move current notable changes to the target (versioned directory) except README.md
find ${SOURCE_DIR} -type f -not -name README.md -exec mv {} ${TARGET_DIR} \;
