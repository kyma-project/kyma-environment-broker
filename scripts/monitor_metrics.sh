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

# Restart the kyma-environment-broker pod
echo "Restarting kyma-environment-broker pod..."
POD_NAME=$(kubectl get pod -l app.kubernetes.io/name=kyma-environment-broker -n kcp-system -o jsonpath='{.items[0].metadata.name}')
kubectl delete pod $POD_NAME -n kcp-system

# Check if kyma-environment-broker pod is in READY state
echo "Waiting for kyma-environment-broker pod to be in READY state..."
kubectl wait --namespace kcp-system --for=condition=Ready pod -l app.kubernetes.io/name=kyma-environment-broker --timeout=60s
EXIT_CODE=$?
if [ $EXIT_CODE -ne 0 ]; then
  echo "The kyma-environment-broker pod did not become READY within the timeout."
  echo "Fetching the logs from the pod..."
  POD_NAME=$(kubectl get pod -l app.kubernetes.io/name=kyma-environment-broker -n kcp-system -o jsonpath='{.items[0].metadata.name}')
  kubectl logs $POD_NAME -n kcp-system
  exit 1
fi

kubectl port-forward -n kcp-system deployment/kcp-kyma-environment-broker 8080:8080 5432:5432 &
sleep 5

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

  sleep 2
done