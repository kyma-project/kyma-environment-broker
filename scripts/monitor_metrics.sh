#!/usr/bin/env bash

# This script continuously monitors selected kyma-environment-broker metrics.
# It captures values such as goroutine count, open file descriptors, 
# database connection pool statistics, and memory usage.
# The script appends these metrics as JSON objects to /tmp/keb_metrics.jsonl
# at 10-second intervals.
#
# Usage:
#   ./monitor_metrics.sh
#
# Metrics collected:
# - go_goroutines: Number of current goroutines.
# - process_open_fds: Number of open file descriptors.
# - go_sql_stats_connections_idle: Idle DB connections.
# - go_sql_stats_connections_max_open: Max open DB connections.
# - go_sql_stats_connections_in_use: Active DB connections.
# - go_memstats_alloc_bytes: Allocated memory in MiB.
# - go_memstats_stack_inuse_bytes: Stack memory in use in MiB.
# - go_memstats_heap_inuse_bytes: Heap memory in use in MiB.
#
# Output:
# Each line in /tmp/keb_metrics.jsonl is a JSON object with a timestamp and the collected metrics.

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

METRICS_FILE="/tmp/keb_metrics.jsonl"
touch "$METRICS_FILE"

while true; do
  TIMESTAMP=$(date +%s)
  
  # Try to fetch metrics, skip this iteration if it fails
  if ! METRICS=$(curl -s --max-time 5 http://localhost:30080/metrics 2>/dev/null); then
    echo "Warning: Failed to fetch metrics at $(date), retrying..." >&2
    sleep 2
    continue
  fi
  
  # Skip if metrics are empty
  if [ -z "$METRICS" ]; then
    echo "Warning: Empty metrics response at $(date), retrying..." >&2
    sleep 2
    continue
  fi

  GO_GOROUTINES=$(echo "$METRICS" | awk '/^go_goroutines/ {print $2}')
  OPEN_FDS=$(echo "$METRICS" | awk '/^process_open_fds/ {print $2}')
  DB_IDLE=$(echo "$METRICS" | awk '/go_sql_stats_connections_idle\{db_name="broker"\}/ {print $2}')
  DB_MAX_OPEN=$(echo "$METRICS" | awk '/go_sql_stats_connections_max_open\{db_name="broker"\}/ {print $2}')
  DB_IN_USE=$(echo "$METRICS" | awk '/go_sql_stats_connections_in_use\{db_name="broker"\}/ {print $2}')
  MEM_ALLOC=$(echo "$METRICS" | awk '/^go_memstats_alloc_bytes / {printf "%.2f", $2/1048576}')
  MEM_STACK=$(echo "$METRICS" | awk '/^go_memstats_stack_inuse_bytes/ {printf "%.2f", $2/1048576}')
  MEM_HEAP=$(echo "$METRICS" | awk '/^go_memstats_heap_inuse_bytes/ {printf "%.2f", $2/1048576}')

  echo "{\"timestamp\": $TIMESTAMP, \"goroutines\": $GO_GOROUTINES, \"open_fds\": $OPEN_FDS, \"db_idle\": $DB_IDLE, \"db_max_open\": $DB_MAX_OPEN, \"db_in_use\": $DB_IN_USE, \"mem_alloc\": $MEM_ALLOC, \"mem_stack\": $MEM_STACK, \"mem_heap\": $MEM_HEAP}" >> "$METRICS_FILE"

  sleep 2
done