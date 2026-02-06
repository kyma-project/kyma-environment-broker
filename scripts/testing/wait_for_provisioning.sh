
#!/usr/bin/env bash
# Wait for provisioning to finish
# Usage: wait_for_provisioning.sh <instance_id>

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
#set -o pipefail # prevents errors in a pipeline from being masked

INSTANCE_ID=${1:?Instance ID required}
while true; do
  STATUS=$(curl --request GET \
    --url http://localhost:8080/runtimes \
    --header 'Content-Type: application/json' \
    --header 'X-Broker-API-Version: 2.16' | jq -r --arg iid "$INSTANCE_ID" '.data[] | select(.instanceID==$iid) | .status.provisioning.state')

  echo "Current provisioning status: $STATUS"
  if [ "$STATUS" == "succeeded" ]; then
    echo "Provisioning succeeded."
    break
  elif [ "$STATUS" == "failed" ]; then
    echo "Provisioning failed."
    exit 1
  fi
  sleep 5
done
