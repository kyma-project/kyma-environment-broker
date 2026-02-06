#!/usr/bin/env bash
# Assert the total count of provisioned instances
# Usage: assert_total_count.sh <expected_count> <plan> [api_url]

set -o nounset
set -o errexit
set -E
set -o pipefail

EXPECTED=${1:?Expected count required}
PLAN=${2:?Plan name required}
API_URL=${3:-http://localhost:30080}

TOTAL_COUNT=$(curl --request GET \
  --url "$API_URL/runtimes?plan=$PLAN" \
  --header 'Content-Type: application/json' \
  --header 'X-Broker-API-Version: 2.16' | jq .totalCount)

if [ "$TOTAL_COUNT" -eq "$EXPECTED" ]; then
  echo "Assertion passed: totalCount is $TOTAL_COUNT"
else
  echo "Assertion failed: totalCount is not $EXPECTED. Actual value: $TOTAL_COUNT"
  exit 1
fi
