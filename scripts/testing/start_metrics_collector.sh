#!/usr/bin/env bash
# Start the metrics collector in the background
# Usage: start_metrics_collector.sh

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

nohup bash scripts/monitor_metrics.sh > /dev/null 2>&1 &
echo $! > /tmp/metrics_pid
echo "Metrics collector started with PID $(cat /tmp/metrics_pid)"
