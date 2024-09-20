#!/bin/bash -x

set -o nounset  # treat unset variables as an error and exit immediately.
set -e # exit immediately when a command fails.

DIR=$1

cd "$(dirname "$0")/bin"

find ${DIR} -exec chmod u+w {} \;
rm -rf ${DIR}
