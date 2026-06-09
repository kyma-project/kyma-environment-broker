
#!/usr/bin/env bash
# Wait for updating to finish
# Usage: wait_for_update.sh <instance_id> [base_url]

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

INSTANCE_ID=${1:?Instance ID required}
BASE_URL=${2:-http://localhost:8080}

while true; do
  STATUS=$(curl --request GET \
    --url "${BASE_URL}/runtimes" \
    --header 'Content-Type: application/json' \
    --header 'X-Broker-API-Version: 2.16' | jq -r --arg iid "$INSTANCE_ID" '.data[] | select(.instanceID==$iid) | .status.update.data[0].state')

  echo "Current updating status: $STATUS"
  if [ "$STATUS" == "succeeded" ]; then
    echo "Updating succeeded."
    break
  elif [ "$STATUS" == "failed" ]; then
    echo "Updating failed."
    exit 1
  fi
  sleep 5
done
