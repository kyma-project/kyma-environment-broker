#!/usr/bin/env bash
# Collect and summarize provisioning metrics
# Usage: collect_provisioning_metrics.sh <plan_id>

set -o nounset
set -o errexit
set -E
set -o pipefail

PLAN_ID=${1:?Plan ID required}

METRICS=$(curl -s http://localhost:30080/metrics)

SUCCEEDED_TOTAL=$(echo "$METRICS" | grep "kcp_keb_v2_operations_provisioning_succeeded_total{plan_id=\"$PLAN_ID\"}" | awk '{print $2}')
SUCCEEDED_TOTAL=${SUCCEEDED_TOTAL:-0}
FAILED_TOTAL=$(echo "$METRICS" | grep "kcp_keb_v2_operations_provisioning_failed_total{plan_id=\"$PLAN_ID\"}" | awk '{print $2}')
FAILED_TOTAL=${FAILED_TOTAL:-0}
TOTAL=$(echo "$SUCCEEDED_TOTAL + $FAILED_TOTAL" | bc)
if [ "$TOTAL" -eq 0 ]; then
  SUCCESS_RATE=0
else
  SUCCESS_RATE=$(awk "BEGIN {printf \"%.2f\", ($SUCCEEDED_TOTAL / $TOTAL) * 100}")
fi
echo "Success rate of provisioning requests: $SUCCESS_RATE%" >> $GITHUB_STEP_SUMMARY

PROVISIONING_DURATION=$(echo "$METRICS" | grep "kcp_keb_v2_provisioning_duration_minutes_sum{plan_id=\"$PLAN_ID\"}" | awk '{print $2}')
PROVISIONING_DURATION=${PROVISIONING_DURATION:-0}
PROVISIONING_COUNT=$(echo "$METRICS" | grep "kcp_keb_v2_provisioning_duration_minutes_count{plan_id=\"$PLAN_ID\"}" | awk '{print $2}')
PROVISIONING_COUNT=${PROVISIONING_COUNT:-0}
if [ "$PROVISIONING_COUNT" -eq 0 ]; then
  AVG_PROVISIONING_DURATION=0
else
  AVG_PROVISIONING_DURATION=$(awk "BEGIN {printf \"%.2f\", ($PROVISIONING_DURATION / $PROVISIONING_COUNT)}")
fi
echo "Average duration of provisioning requests: $AVG_PROVISIONING_DURATION minutes" >> $GITHUB_STEP_SUMMARY

if [[ "$SUCCESS_RATE" != "100.00" ]]; then
  echo "Provisioning success rate is not 100%"
  exit 1
fi
