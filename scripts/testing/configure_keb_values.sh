#!/usr/bin/env bash
# Configure KEB values.yaml with worker amounts and processing times
# Usage: configure_keb_values.sh [provisioning_time] [provisioning_workers] [update_time] [update_workers] [deprovisioning_time] [deprovisioning_workers]

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

PROVISIONING_TIME=${1:-}
PROVISIONING_WORKERS=${2:-}
UPDATE_TIME=${3:-}
UPDATE_WORKERS=${4:-}
DEPROVISIONING_TIME=${5:-}
DEPROVISIONING_WORKERS=${6:-}

VALUES_FILE="resources/keb/values.yaml"

if [[ -n "$PROVISIONING_TIME" ]]; then
  yq e ".provisioning.maxStepProcessingTime = \"$PROVISIONING_TIME\"" -i "$VALUES_FILE"
fi

if [[ -n "$PROVISIONING_WORKERS" ]]; then
  yq e ".provisioning.workersAmount = $PROVISIONING_WORKERS" -i "$VALUES_FILE"
fi

if [[ -n "$UPDATE_TIME" ]]; then
  yq e ".update.maxStepProcessingTime = \"$UPDATE_TIME\"" -i "$VALUES_FILE"
fi

if [[ -n "$UPDATE_WORKERS" ]]; then
  yq e ".update.workersAmount = $UPDATE_WORKERS" -i "$VALUES_FILE"
fi

if [[ -n "$DEPROVISIONING_TIME" ]]; then
  yq e ".deprovisioning.maxStepProcessingTime = \"$DEPROVISIONING_TIME\"" -i "$VALUES_FILE"
fi

if [[ -n "$DEPROVISIONING_WORKERS" ]]; then
  yq e ".deprovisioning.workersAmount = $DEPROVISIONING_WORKERS" -i "$VALUES_FILE"
fi
