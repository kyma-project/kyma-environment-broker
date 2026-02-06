#!/usr/bin/env bash
# Wait until all operations complete (instances reach succeeded or failed state)
# Usage: wait_for_all_operations.sh <plan_name> <expected_count> <operation_type> [base_url]
# operation_type can be: provisioning, update, deprovisioning

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

PLAN_NAME=${1:?Plan name required}
EXPECTED_COUNT=${2:?Expected count required}
OPERATION_TYPE=${3:?Operation type required (provisioning|update|deprovisioning)}
BASE_URL=${4:-http://localhost:30080}

case "$OPERATION_TYPE" in
  provisioning|update)
    while true; do
      UPDATED_COUNT=$(curl --request GET \
        --url "${BASE_URL}/runtimes?plan=${PLAN_NAME}&state=succeeded,failed" \
        --header 'Content-Type: application/json' \
        --header 'X-Broker-API-Version: 2.16' | jq .totalCount)

      if [ "$UPDATED_COUNT" -eq "$EXPECTED_COUNT" ]; then
        echo "All instances are ${OPERATION_TYPE} complete. Done."
        break
      fi

      sleep 10
    done
    ;;
  deprovisioning)
    while true; do
      DEPROVISIONING_COUNT=$(curl --request GET \
        --url "${BASE_URL}/runtimes?plan=${PLAN_NAME}&state=deprovisioning" \
        --header 'Content-Type: application/json' \
        --header 'X-Broker-API-Version: 2.16' | jq .totalCount)

      if [ "$DEPROVISIONING_COUNT" -eq 0 ]; then
        echo "All instances are deprovisioned. Done."
        break
      fi

      sleep 10
    done
    ;;
  *)
    echo "Error: Unknown operation type: $OPERATION_TYPE"
    exit 1
    ;;
esac
