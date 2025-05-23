#!/usr/bin/env bash

set -o nounset
set -o errexit
set -o pipefail

METRICS_FILE="/tmp/keb_metrics.jsonl"
echo "" > "$METRICS_FILE"

while true; do
  TIMESTAMP=$(date +%s)
  METRICS=$(curl -s http://localhost:8080/metrics)

  GO_GOROUTINES=$(echo "$METRICS" | grep '^go_goroutines' | awk '{print $2}')
  OPEN_FDS=$(echo "$METRICS" | grep '^process_open_fds' | awk '{print $2}')
  DB_IDLE=$(echo "$METRICS" | grep 'go_sql_stats_connections_idle{db_name="broker"}' | awk '{print $2}')
  DB_MAX_OPEN=$(echo "$METRICS" | grep 'go_sql_stats_connections_max_open{db_name="broker"}' | awk '{print $2}')
  DB_IN_USE=$(echo "$METRICS" | grep 'go_sql_stats_connections_in_use{db_name="broker"}' | awk '{print $2}')
  MEM_ALLOC=$(echo "$METRICS" | grep -w '^go_memstats_alloc_bytes' | awk '{printf "%.2f", $2/1048576}')
  MEM_STACK=$(echo "$METRICS" | grep '^go_memstats_stack_inuse_bytes' | awk '{printf "%.2f", $2/1048576}')
  MEM_HEAP=$(echo "$METRICS" | grep '^go_memstats_heap_inuse_bytes' | awk '{printf "%.2f", $2/1048576}')

  echo "{\"timestamp\": $TIMESTAMP, \"goroutines\": $GO_GOROUTINES, \"open_fds\": $OPEN_FDS, \"db_idle\": $DB_IDLE, \"db_max_open\": $DB_MAX_OPEN, \"db_in_use\": $DB_IN_USE, \"mem_alloc\": $MEM_ALLOC, \"mem_stack\": $MEM_STACK, \"mem_heap\": $MEM_HEAP}" >> "$METRICS_FILE"

  sleep 10
done