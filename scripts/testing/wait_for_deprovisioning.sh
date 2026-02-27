
#!/usr/bin/env bash
# Wait for deprovisioning to finish
# Usage: wait_for_deprovisioning.sh <instance_id> <runtime_id>

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

INSTANCE_ID=${1:?Instance ID required}
RUNTIME_ID=${2:?Runtime ID required}

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
if kubectl get runtime "$RUNTIME_ID" -n kcp-system &> /dev/null; then
  echo "RuntimeCR $RUNTIME_ID still exists."
  exit 1
else
  echo "RuntimeCR $RUNTIME_ID successfully removed."
fi
