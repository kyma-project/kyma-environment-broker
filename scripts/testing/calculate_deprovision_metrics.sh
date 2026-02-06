#!/usr/bin/env bash
# Calculate and report deprovisioning success rate and average duration
# Usage: calculate_deprovision_metrics.sh <plan_id> [base_url]

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

PLAN_ID=${1:?Plan ID required}
BASE_URL=${2:-http://localhost:30080}

sleep 15

METRICS=$(curl -s "${BASE_URL}/metrics")

SUCCEEDED_TOTAL=$(echo "$METRICS" | grep "kcp_keb_v2_operations_deprovisioning_succeeded_total{plan_id=\"${PLAN_ID}\"}" | awk '{print $2}')
SUCCEEDED_TOTAL=${SUCCEEDED_TOTAL:-0}
FAILED_TOTAL=$(echo "$METRICS" | grep "kcp_keb_v2_operations_deprovisioning_failed_total{plan_id=\"${PLAN_ID}\"}" | awk '{print $2}')
FAILED_TOTAL=${FAILED_TOTAL:-0}
TOTAL=$(echo "$SUCCEEDED_TOTAL + $FAILED_TOTAL" | bc)

if [ "$TOTAL" -eq 0 ]; then
  SUCCESS_RATE=0
else
  SUCCESS_RATE=$(awk "BEGIN {printf \"%.2f\", ($SUCCEEDED_TOTAL / $TOTAL) * 100}")
fi

echo "Success rate of deprovisioning requests: $SUCCESS_RATE%" >> $GITHUB_STEP_SUMMARY

DEPROVISIONING_DURATION=$(echo "$METRICS" | grep "kcp_keb_v2_deprovisioning_duration_minutes_sum{plan_id=\"${PLAN_ID}\"}" | awk '{print $2}')
DEPROVISIONING_DURATION=${DEPROVISIONING_DURATION:-0}
DEPROVISIONING_COUNT=$(echo "$METRICS" | grep "kcp_keb_v2_deprovisioning_duration_minutes_count{plan_id=\"${PLAN_ID}\"}" | awk '{print $2}')
DEPROVISIONING_COUNT=${DEPROVISIONING_COUNT:-0}

if [ "$DEPROVISIONING_COUNT" -eq 0 ]; then
  AVG_DEPROVISIONING_DURATION=0
else
  AVG_DEPROVISIONING_DURATION=$(awk "BEGIN {printf \"%.2f\", ($DEPROVISIONING_DURATION / $DEPROVISIONING_COUNT)}")
fi

echo "Average duration of deprovisioning requests: $AVG_DEPROVISIONING_DURATION minutes" >> $GITHUB_STEP_SUMMARY

if [[ "$SUCCESS_RATE" != "100.00" ]]; then
  echo "Error: SUCCESS_RATE is $SUCCESS_RATE, expected 100.00"
  exit 1
fi

echo "Deprovisioning success rate: $SUCCESS_RATE%"
echo "Average deprovisioning duration: $AVG_DEPROVISIONING_DURATION minutes"
