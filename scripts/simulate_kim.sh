#!/usr/bin/env bash

set -o nounset
set -o errexit
set -E
set -o pipefail

get_provisioning_runtimes() {
  curl --request GET \
    --url http://localhost:8080/runtimes?state=provisioning \
    --header 'Content-Type: application/json' \
    --header 'X-Broker-API-Version: 2.16' | jq .totalCount
}

COUNT=$(get_provisioning_runtimes)
echo "Initial provisioning runtimes count: $COUNT"

while (( COUNT > 0 )); do
  RUNTIMES=$(kubectl get runtimes -n kcp-system -o json | jq -r '.items[] | select(.status.state != "Ready") | .metadata.name')

  for RUNTIME_ID in $RUNTIMES; do
    kubectl patch runtime "$RUNTIME_ID" \
      -n kcp-system \
      --type merge \
      --subresource status \
      -p '{"status": {"state": "Ready"}}'
  done
  
  sleep 10
  
  COUNT=$(get_provisioning_runtimes)
  if (( COUNT == 0 )); then
    echo "All runtimes are ready. Done."
    break
  fi
  echo "Provisioning runtimes remaining: $COUNT"
done
