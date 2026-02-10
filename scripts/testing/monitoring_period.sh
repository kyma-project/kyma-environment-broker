#!/usr/bin/env bash
# Wait for a monitoring period (in minutes)
# Usage: monitoring_period.sh <minutes> [label]

set -o nounset
set -o errexit
set -E
set -o pipefail

MINUTES=${1:?Monitoring period in minutes required}
LABEL=${2:-}

if [ -n "$LABEL" ]; then
  echo "$LABEL monitoring for $MINUTES minutes..."
else
  echo "Monitoring for $MINUTES minutes..."
fi
sleep $((MINUTES * 60))
echo "Monitoring completed"
