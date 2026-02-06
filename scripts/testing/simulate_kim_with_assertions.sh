#!/usr/bin/env bash
# Simulate KIM and verify expected number of succeeded instances
# Usage: simulate_kim_with_assertions.sh <plan_name> <expected_count> [base_url] [kim_delay_seconds]

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

PLAN_NAME=${1:?Plan name required}
EXPECTED_COUNT=${2:?Expected count required}
BASE_URL=${3:-http://localhost:30080}
KIM_DELAY_SECONDS=${4:-${KIM_DELAY_SECONDS:-0}}

export KIM_DELAY_SECONDS
scripts/simulate_kim.sh

RESPONSE_JSON=$(curl --request GET \
  --url "${BASE_URL}/runtimes?plan=${PLAN_NAME}&state=succeeded" \
  --header 'Content-Type: application/json' \
  --header 'X-Broker-API-Version: 2.16')
SUCCEEDED_TOTAL_COUNT=$(echo "$RESPONSE_JSON" | jq .totalCount)

if [ "$SUCCEEDED_TOTAL_COUNT" -eq "$EXPECTED_COUNT" ]; then
  echo "Assertion passed: succeeded totalCount is $SUCCEEDED_TOTAL_COUNT"
else
  echo "Assertion failed: succeeded totalCount is not $EXPECTED_COUNT. Actual value: $SUCCEEDED_TOTAL_COUNT"
  exit 1
fi
