#!/usr/bin/env bash
# Calculate and report update success rate
# Usage: calculate_update_success_rate.sh <plan_id> [base_url]

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

PLAN_ID=${1:?Plan ID required}
BASE_URL=${2:-http://localhost:30080}

sleep 15

METRICS=$(curl -s "${BASE_URL}/metrics")

echo "DEBUG: Raw metrics fetched from ${BASE_URL}/metrics:"
echo "$METRICS"

set +o pipefail
SUCCEEDED_TOTAL=$(echo "$METRICS" | grep "kcp_keb_v2_operation_result.*plan_id=\"${PLAN_ID}\".*state=\"succeeded\".*type=\"update\"" | wc -l | xargs)
SUCCEEDED_TOTAL=${SUCCEEDED_TOTAL:-0}
FAILED_TOTAL=$(echo "$METRICS" | grep "kcp_keb_v2_operation_result.*plan_id=\"${PLAN_ID}\".*state=\"failed\".*type=\"update\"" | wc -l | xargs)
FAILED_TOTAL=${FAILED_TOTAL:-0}
set -o pipefail

echo "DEBUG: SUCCEEDED_TOTAL=$SUCCEEDED_TOTAL"
echo "DEBUG: FAILED_TOTAL=$FAILED_TOTAL"

TOTAL=$(echo "$SUCCEEDED_TOTAL + $FAILED_TOTAL" | bc)

echo "DEBUG: SUCCEEDED_TOTAL=$SUCCEEDED_TOTAL, FAILED_TOTAL=$FAILED_TOTAL, TOTAL=$TOTAL"

if [ "$TOTAL" -eq 0 ]; then
  SUCCESS_RATE=0
else
    SUCCESS_RATE=$(awk "BEGIN {printf \"%.2f\", ($SUCCEEDED_TOTAL / $TOTAL) * 100}")
fi

echo "Success rate of update requests: $SUCCESS_RATE%" >> $GITHUB_STEP_SUMMARY

if [[ "$SUCCESS_RATE" != "100.00" ]]; then
  echo "Error: SUCCESS_RATE is $SUCCESS_RATE, expected 100.00"
  exit 1
fi

echo "Update success rate: $SUCCESS_RATE%"
