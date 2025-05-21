#!/usr/bin/env bash

set -o nounset
set -o errexit
set -E
set -o pipefail

AGE_THRESHOLD_SECONDS=60

get_provisioning_runtimes() {
  local count
  if ! count=$(curl --silent --fail --request GET \
      --url http://localhost:8080/runtimes?state=provisioning \
      --header 'Content-Type: application/json' \
      --header 'X-Broker-API-Version: 2.16' | jq .totalCount); then
    echo "Warning: Failed to fetch provisioning runtimes. Assuming at least 1 remains." >&2
    echo 1
  else
    echo "$count"
  fi
}

is_older_than_threshold() {
  local creation_timestamp="$1"
  local creation_seconds
  local now_seconds

  creation_seconds=$(date --date="$creation_timestamp" +%s)
  now_seconds=$(date +%s)

  local age_seconds=$(( now_seconds - creation_seconds ))
  (( age_seconds >= AGE_THRESHOLD_SECONDS ))
}

COUNT=$(get_provisioning_runtimes)
echo "Initial provisioning runtimes count: $COUNT"

while (( COUNT > 0 )); do
  RUNTIMES=$(kubectl get runtimes -n kcp-system -o json | jq -r \
    '.items[] | select(.status.state != "Ready") | "\(.metadata.name) \(.metadata.creationTimestamp)"')

  while read -r RUNTIME_ID CREATION_TS; do
    if is_older_than_threshold "$CREATION_TS"; then
      echo "Patching runtime: $RUNTIME_ID (created at $CREATION_TS)"
      kubectl patch runtime "$RUNTIME_ID" \
        -n kcp-system \
        --type merge \
        --subresource status \
        -p '{"status": {"state": "Ready"}}'
    else
      echo "Skipping $RUNTIME_ID — too recent (created at $CREATION_TS)"
    fi
  done <<< "$RUNTIMES"

  sleep 20

  COUNT=$(get_provisioning_runtimes)
  if (( COUNT == 0 )); then
    echo "All runtimes are ready. Done."
    break
  fi
  echo "Provisioning runtimes remaining: $COUNT"
done
