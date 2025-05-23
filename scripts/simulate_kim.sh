#!/usr/bin/env bash

# This script simulates the successful readiness of runtimes stuck in
# a provisioning state by patching them to "Ready" if they are older than
# a specified threshold.
# It has the following arguments:
#   - KIM delay seconds (default: 60 seconds)
# ./simulate_kim.sh 120

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

KIM_DELAY_SECONDS="${KIM_DELAY_SECONDS:-${1:-60}}"

GO_GOROUTINES_ARRAY=()

PROCESS_OPEN_FDS_ARRAY=()

DB_CONNECTIONS_IDLE_ARRAY=()
DB_CONNECTIONS_MAX_OPEN_ARRAY=()
DB_CONNECTIONS_IN_USE_ARRAY=()

MEM_ALLOC_BYTES_ARRAY=()
MEM_STACK_INUSE_BYTES_ARRAY=()
MEM_HEAP_INUSE_BYTES_ARRAY=()

get_provisioning_runtimes() {
  local count
  if ! count=$(curl --silent --fail --request GET \
      --url http://localhost:8080/runtimes?state=provisioning \
      --header 'Content-Type: application/json' \
      --header 'X-Broker-API-Version: 2.16' | jq .totalCount); then
    echo "Warning: Failed to fetch provisioning runtimes. Assuming at least 1 remains." >&2
    echo 1
  else
    echo "$count"
  fi
}

is_older_than_threshold() {
  local creation_timestamp="$1"
  local creation_seconds
  local now_seconds

  creation_seconds=$(date --date="$creation_timestamp" +%s)
  now_seconds=$(date +%s)

  local age_seconds=$(( now_seconds - creation_seconds ))
  (( age_seconds >= KIM_DELAY_SECONDS ))
}

COUNT=$(get_provisioning_runtimes)
echo "Initial provisioning runtimes count: $COUNT"

while (( COUNT > 0 )); do
  RUNTIMES=$(kubectl get runtimes -n kcp-system -o json | jq -r \
    '.items[] | select(.status.state != "Ready") | "\(.metadata.name) \(.metadata.creationTimestamp)"')

  while read -r RUNTIME_ID CREATION_TS; do
    if [[ -z "$RUNTIME_ID" || -z "$CREATION_TS" ]]; then
      continue
    fi
    if is_older_than_threshold "$CREATION_TS"; then
      echo "Patching runtime: $RUNTIME_ID (created at $CREATION_TS)"
      kubectl patch runtime "$RUNTIME_ID" \
        -n kcp-system \
        --type merge \
        --subresource status \
        -p '{"status": {"state": "Ready"}}'
    fi
  done <<< "$RUNTIMES"

  sleep 10
  
  METRICS=$(curl -s http://localhost:8080/metrics)
  
  GO_GOROUTINES=$(echo "$METRICS" | grep '^go_goroutines' | awk '{print $2}')
  GO_GOROUTINES_ARRAY+=("$GO_GOROUTINES")
  
  PROCESS_OPEN_FDS=$(echo "$METRICS" | grep '^process_open_fds' | awk '{print $2}')
  PROCESS_OPEN_FDS_ARRAY+=("$PROCESS_OPEN_FDS")
  
  DB_CONNECTIONS_IDLE=$(echo "$METRICS" | grep 'go_sql_stats_connections_idle{db_name="broker"}' | awk '{print $2}')
  DB_CONNECTIONS_IDLE_ARRAY+=("$DB_CONNECTIONS_IDLE")
  DB_CONNECTIONS_MAX_OPEN=$(echo "$METRICS" | grep 'go_sql_stats_connections_max_open{db_name="broker"}' | awk '{print $2}')
  DB_CONNECTIONS_MAX_OPEN_ARRAY+=("$DB_CONNECTIONS_MAX_OPEN")
  DB_CONNECTIONS_IN_USE=$(echo "$METRICS" | grep 'go_sql_stats_connections_in_use{db_name="broker"}' | awk '{print $2}')
  DB_CONNECTIONS_IN_USE_ARRAY+=("$DB_CONNECTIONS_IN_USE")
  
  MEM_ALLOC_BYTES=$(echo "$METRICS" | grep -w '^go_memstats_alloc_bytes' | LC_ALL=C awk '{printf "%.2f", $2/1048576}')
  MEM_ALLOC_BYTES_ARRAY+=("$MEM_ALLOC_BYTES")
  MEM_STACK_INUSE_BYTES=$(echo "$METRICS" | grep '^go_memstats_stack_inuse_bytes' | LC_ALL=C awk '{printf "%.2f", $2/1048576}')
  MEM_STACK_INUSE_BYTES_ARRAY+=("$MEM_STACK_INUSE_BYTES")
  MEM_HEAP_INUSE_BYTES=$(echo "$METRICS" | grep '^go_memstats_heap_inuse_bytes' | LC_ALL=C awk '{printf "%.2f", $2/1048576}')
  MEM_HEAP_INUSE_BYTES_ARRAY+=("$MEM_HEAP_INUSE_BYTES")

  COUNT=$(get_provisioning_runtimes)
  if (( COUNT == 0 )); then
    echo "All runtimes are ready. Done."
    break
  fi
  echo "Provisioning runtimes remaining: $COUNT"
done

MERMAID_GO_GOROUTINES=$(IFS=, ; echo "[${GO_GOROUTINES_ARRAY[*]}]")
{
  echo '```mermaid'
  echo "xychart-beta title \"Goroutines\" line $MERMAID_GO_GOROUTINES"
  echo '```'
} >> "$GITHUB_STEP_SUMMARY"

MERMAID_PROCESS_OPEN_FDS=$(IFS=, ; echo "[${PROCESS_OPEN_FDS_ARRAY[*]}]")
{
  echo '```mermaid'
  echo "xychart-beta title \"Open FDs\" line $MERMAID_PROCESS_OPEN_FDS"
  echo '```'
} >> "$GITHUB_STEP_SUMMARY"

MERMAID_DB_CONNECTIONS_IDLE=$(IFS=, ; echo "[${DB_CONNECTIONS_IDLE_ARRAY[*]}]")
MERMAID_DB_CONNECTIONS_MAX_OPEN=$(IFS=, ; echo "[${DB_CONNECTIONS_MAX_OPEN_ARRAY[*]}]")
MERMAID_DB_CONNECTIONS_IN_USE=$(IFS=, ; echo "[${DB_CONNECTIONS_IN_USE_ARRAY[*]}]")
{
  echo '```mermaid'
  echo "xychart-beta title \"DB connections\" line \"Idle\" $MERMAID_DB_CONNECTIONS_IDLE line \"Max open\" $MERMAID_DB_CONNECTIONS_MAX_OPEN line \"In use\" $MERMAID_DB_CONNECTIONS_IN_USE"
  echo '```'
} >> "$GITHUB_STEP_SUMMARY"
echo "<div align=\"center\">" >> "$GITHUB_STEP_SUMMARY"
echo "" >> "$GITHUB_STEP_SUMMARY"
echo "| Color | Type     |" >> "$GITHUB_STEP_SUMMARY"
echo "|-------|----------|" >> "$GITHUB_STEP_SUMMARY"
echo "| Blue  | Idle     |" >> "$GITHUB_STEP_SUMMARY"
echo "| Green | Max open |" >> "$GITHUB_STEP_SUMMARY"
echo "| Red   | In use   |" >> "$GITHUB_STEP_SUMMARY"
echo "</div>" >> "$GITHUB_STEP_SUMMARY"
echo "" >> "$GITHUB_STEP_SUMMARY"

MERMAID_MEM_ALLOC_BYTES=$(IFS=, ; echo "[${MEM_ALLOC_BYTES_ARRAY[*]}]")
MERMAID_MEM_STACK_INUSE_BYTES=$(IFS=, ; echo "[${MEM_STACK_INUSE_BYTES_ARRAY[*]}]")
MERMAID_MEM_HEAP_INUSE_BYTES=$(IFS=, ; echo "[${MEM_HEAP_INUSE_BYTES_ARRAY[*]}]")
{
  echo '```mermaid'
  echo "xychart-beta title \"Go Memstats\" y-axis \"Memory (in MiB)\" line \"Alloc bytes\" $MERMAID_MEM_ALLOC_BYTES line \"Stack in use bytes\" $MERMAID_MEM_STACK_INUSE_BYTES line \"Heap in use bytes\" $MERMAID_MEM_HEAP_INUSE_BYTES"
  echo '```'
} >> "$GITHUB_STEP_SUMMARY"

echo "<div align=\"center\">" >> "$GITHUB_STEP_SUMMARY"
echo "" >> "$GITHUB_STEP_SUMMARY"
echo "| Color | Type               |" >> "$GITHUB_STEP_SUMMARY"
echo "|-------|--------------------|" >> "$GITHUB_STEP_SUMMARY"
echo "| Blue  | Alloc bytes        |" >> "$GITHUB_STEP_SUMMARY"
echo "| Green | Stack in use bytes |" >> "$GITHUB_STEP_SUMMARY"
echo "| Red   | Heap in use bytes  |" >> "$GITHUB_STEP_SUMMARY"
echo "</div>" >> "$GITHUB_STEP_SUMMARY"
echo "" >> "$GITHUB_STEP_SUMMARY"