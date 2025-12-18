#!/usr/bin/env bash

# Check KEB logs for errors and warnings
# Usage: check-keb-logs.sh <pod_name> <stage> [allow_apiserver_error]
#   pod_name: Name of the KEB pod
#   stage: Stage name for logging (e.g., "provisioning", "update", "deprovisioning")
#   allow_apiserver_error: Optional, set to "true" to allow APIServerURL validation error

POD_NAME="${1:?Pod name is required}"
STAGE="${2:?Stage name is required}"
ALLOW_APISERVER_ERROR="${3:-false}"

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo "Checking KEB logs after ${STAGE}..."

LOGS=$(kubectl logs -n kcp-system "$POD_NAME")

# Get all errors
ERRORS=$(echo "$LOGS" | grep -E '"level":"ERROR"' || true)

# Filter out expected error if allowed
if [ "$ALLOW_APISERVER_ERROR" == "true" ]; then
  ERRORS=$(echo "$ERRORS" | grep -v "while getting APIServerURL: while validation kubeconfig fetched by provisioner: there are no cluster certificate or server info" || true)
fi

WARNINGS=$(echo "$LOGS" | grep -E '"level":"WARNING"' || true)

if [ -n "$ERRORS" ]; then
  if [ "$ALLOW_APISERVER_ERROR" == "true" ]; then
    echo "Unexpected errors found in logs:"
  else
    echo "Errors found in logs:"
  fi
  echo "$ERRORS"
  exit 1
fi

if [ -n "$WARNINGS" ]; then
  echo "Warnings found in logs:"
  echo "$WARNINGS"
  exit 1
fi

if [ "$ALLOW_APISERVER_ERROR" == "true" ]; then
  echo "No unexpected errors or warnings found in logs."
else
  echo "No errors or warnings found in logs."
fi
