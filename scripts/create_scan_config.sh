#!/usr/bin/env bash

# This script has the following arguments:
#                       binary image reference (mandatory)
#                       filename of file to be created (optional)
#                       release tag (optional)
# ./create_scan_config image temp_scan_congitfig.yaml          - use when building module image
# ./create_scan_config image temp_scan_congitfig.yaml tag      - use when bumping the config on the main branch

FILENAME=${1-../sec-scanners-config.yaml}
TAG=${2:-}

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being maskedPORT=5001

echo "Creating security scan configuration file:"

# add rc-tag when creating the config on the main branch
if [ -n "${TAG}" ]; then
  cat <<EOF | tee ${FILENAME}
# this file is autogenerated (scripts/create_scan_config.sh), do not modify
module-name: kyma-environment-broker
rc-tag: ${TAG}
protecode:
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-broker:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-deprovision-retrigger-job:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environments-cleanup-job:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-runtime-reconciler:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-trial-cleanup-job:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-subscription-cleanup-job:${TAG}
whitesource:
  language: golang-mod
  subprojects: false
  exclude:
    - "**/*_test.go"
    - "testing/**"
EOF
else
  cat <<EOF | tee ${FILENAME}
module-name: btp-operator
protecode:
  - ${IMAGE}
whitesource:
  language: golang-mod
  subprojects: false
  exclude:
    - "**/*_test.go"
    - "testing/**
EOF
fi
