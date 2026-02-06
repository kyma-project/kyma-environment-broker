#!/usr/bin/env bash
# Assert runtime metrics match expected values
# Usage: 
#   For plan totalCount: assert_runtime_count.sh <plan_name> <expected_count> [base_url]
#   For instance update count: assert_runtime_count.sh <instance_id> <expected_count> [base_url] update
#
# Examples:
#   assert_runtime_count.sh azure 10 http://localhost:30080
#   assert_runtime_count.sh azure-cluster 1 http://localhost:8080 update

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

IDENTIFIER=${1:?Plan name or instance ID required}
EXPECTED_COUNT=${2:?Expected count required}
BASE_URL=${3:-http://localhost:30080}
MODE=${4:-plan}

if [ "$MODE" = "update" ]; then
  # Instance update count mode
  INSTANCE_ID="$IDENTIFIER"
  TOTAL_COUNT=$(curl --request GET \
    --url "${BASE_URL}/runtimes" \
    --header 'Content-Type: application/json' \
    --header 'X-Broker-API-Version: 2.16' | jq -r --arg iid "$INSTANCE_ID" '.data[] | select(.instanceID==$iid) | .status.update.totalCount')
  
  if [ "$TOTAL_COUNT" -eq "$EXPECTED_COUNT" ]; then
    echo "Assertion passed: update totalCount for instance $INSTANCE_ID is $TOTAL_COUNT"
  else
    echo "Assertion failed: update totalCount for instance $INSTANCE_ID is not $EXPECTED_COUNT. Actual value: $TOTAL_COUNT"
    exit 1
  fi
else
  # Plan totalCount mode
  PLAN_NAME="$IDENTIFIER"
  TOTAL_COUNT=$(curl --request GET \
    --url "${BASE_URL}/runtimes?plan=${PLAN_NAME}" \
    --header 'Content-Type: application/json' \
    --header 'X-Broker-API-Version: 2.16' | jq .totalCount)

  if [ "$TOTAL_COUNT" -eq "$EXPECTED_COUNT" ]; then
    echo "Assertion passed: totalCount for plan $PLAN_NAME is $TOTAL_COUNT"
  else
    echo "Assertion failed: totalCount for plan $PLAN_NAME is not $EXPECTED_COUNT. Actual value: $TOTAL_COUNT"
    exit 1
  fi
fi
