#!/usr/bin/env bash
# Wait for instance to reach succeeded state
# Usage: wait_for_instance_state.sh <instance_id> [base_url]

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
#set -o pipefail # prevents errors in a pipeline from being masked

INSTANCE_ID=${1:?Instance ID required}
BASE_URL=${2:-http://localhost:8080}

while true; do
  INSTANCE_STATE=$(curl --request GET \
    --url "${BASE_URL}/runtimes?instance_id=${INSTANCE_ID}" \
    --header 'Content-Type: application/json' \
    --header 'X-Broker-API-Version: 2.16' | jq -r '.data[0].status.state')

  echo "Current instance state: $INSTANCE_STATE"
  
  if [ "$INSTANCE_STATE" = "succeeded" ]; then
    echo "Instance state is succeeded."
    break
  elif [ "$INSTANCE_STATE" = "failed" ]; then
    echo "Instance state is failed."
    exit 1
  fi
  
  sleep 5
done
