#!/usr/bin/env bash

# This script simulates the successful readiness of runtimes stuck in
# a provisioning state by patching them to "Ready" if they are older than
# a specified threshold.
# It has the following arguments:
#   - KIM delay seconds (default: 60 seconds)
# ./simulate_kim.sh 120

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

KIM_DELAY_SECONDS="${KIM_DELAY_SECONDS:-${1:-60}}"

get_provisioning_runtimes() {
  curl --request GET \
    --url http://localhost:30080/runtimes?state=provisioning \
    --header 'Content-Type: application/json' \
    --header 'X-Broker-API-Version: 2.16' | jq .totalCount
}

is_older_than_threshold() {
  local creation_timestamp="$1"
  local creation_seconds
  local now_seconds

  creation_seconds=$(date --date="$creation_timestamp" +%s)
  now_seconds=$(date +%s)

  local age_seconds=$(( now_seconds - creation_seconds ))
  (( age_seconds >= KIM_DELAY_SECONDS ))
}

COUNT=$(get_provisioning_runtimes)
echo "Initial provisioning runtimes count: $COUNT"

while (( COUNT > 0 )); do
  echo "Provisioning runtimes remaining: $COUNT"
  RUNTIMES=$(kubectl get runtimes -n kcp-system -o json | jq -r \
    '.items[] | select(.status.state != "Ready") | "\(.metadata.name) \(.metadata.creationTimestamp)"')

  while read -r RUNTIME_ID CREATION_TS; do
    if [[ -z "$RUNTIME_ID" || -z "$CREATION_TS" ]]; then
      continue
    fi
    if is_older_than_threshold "$CREATION_TS"; then
      echo "Patching runtime: $RUNTIME_ID (created at $CREATION_TS)"
      kubectl patch runtime "$RUNTIME_ID" \
        -n kcp-system \
        --type merge \
        --subresource status \
        -p '{"status": {"state": "Ready"}}'
    fi
  done <<< "$RUNTIMES"

  sleep 10

  COUNT=$(get_provisioning_runtimes)
done

echo "All runtimes are ready. Done."
