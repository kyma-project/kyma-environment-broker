
#!/usr/bin/env bash
# Wait for deprovisioning to finish
# Usage: wait_for_deprovisioning.sh <instance_id>

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

INSTANCE_ID=${1:?Instance ID required}
while true; do
  RESULT=$(curl --request GET \
    --url http://localhost:8080/runtimes \
    --header 'Content-Type: application/json' \
    --header 'X-Broker-API-Version: 2.16' | jq -r --arg iid "$INSTANCE_ID" '.data[] | select(.instanceID==$iid)')

  if [ -z "$RESULT" ]; then
    echo "Deprovisioning succeeded."
    break
  fi
  sleep 5
done

# Check if RuntimeCR was removed
RUNTIME_COUNT=$(kubectl get runtimes -n kcp-system | wc -l)
if [ "$RUNTIME_COUNT" -ne 0 ]; then
  echo "RuntimeCR still exists."
  exit 1
fi
