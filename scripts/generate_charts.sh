#!/usr/bin/env bash

# This script aggregates kyma-environment-broker metrics (such as goroutines,
# file descriptors, memory usage, and database connections) from a JSONL file
# and generates visual summaries using Mermaid charts for GitHub Actions.

# Usage:
#   ./generate_charts.sh

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

sleep 20

kill $(cat /tmp/metrics_pid) || echo "Metrics script not running"
METRICS_FILE="/tmp/keb_metrics.jsonl"

jq -s '
{
  goroutines: map(.goroutines),
  open_fds: map(.open_fds),
  db_idle: map(.db_idle),
  db_max_open: map(.db_max_open),
  db_in_use: map(.db_in_use),
  mem_alloc: map(.mem_alloc),
  mem_stack: map(.mem_stack),
  mem_heap: map(.mem_heap)
}' "$METRICS_FILE" > /tmp/aggregated_metrics.json
      
echo '```mermaid' >> $GITHUB_STEP_SUMMARY
echo "xychart-beta title \"Goroutines\" line \"Goroutines\" [$(jq -r '.goroutines | @csv' /tmp/aggregated_metrics.json)]" >> $GITHUB_STEP_SUMMARY
echo '```' >> $GITHUB_STEP_SUMMARY

echo '```mermaid' >> $GITHUB_STEP_SUMMARY
echo "xychart-beta title \"Open FDs\" line \"open_fds\" [$(jq -r '.open_fds | @csv' /tmp/aggregated_metrics.json)]" >> $GITHUB_STEP_SUMMARY
echo '```' >> $GITHUB_STEP_SUMMARY

echo '```mermaid' >> $GITHUB_STEP_SUMMARY
echo "xychart-beta title \"Go Memstats\" y-axis \"Memory (in MiB)\" line \"Alloc\" [$(jq -r '.mem_alloc | @csv' /tmp/aggregated_metrics.json)] line \"Heap\" [$(jq -r '.mem_heap | @csv' /tmp/aggregated_metrics.json)] line \"Stack\" [$(jq -r '.mem_stack | @csv' /tmp/aggregated_metrics.json)]" >> $GITHUB_STEP_SUMMARY
echo '```' >> $GITHUB_STEP_SUMMARY
echo "<div align=\"center\">" >> "$GITHUB_STEP_SUMMARY"
echo "" >> "$GITHUB_STEP_SUMMARY"
echo "| Color | Type               |" >> "$GITHUB_STEP_SUMMARY"
echo "|-------|--------------------|" >> "$GITHUB_STEP_SUMMARY"
echo "| Green | Heap in use bytes  |" >> "$GITHUB_STEP_SUMMARY"
echo "| Blue  | Alloc bytes        |" >> "$GITHUB_STEP_SUMMARY"
echo "| Red   | Stack in use bytes |" >> "$GITHUB_STEP_SUMMARY"
echo "</div>" >> "$GITHUB_STEP_SUMMARY"
echo "" >> "$GITHUB_STEP_SUMMARY"

echo '```mermaid' >> $GITHUB_STEP_SUMMARY
echo "xychart-beta title \"DB Connections\" line \"Idle\" [$(jq -r '.db_idle | @csv' /tmp/aggregated_metrics.json)] line \"In Use\" [$(jq -r '.db_in_use | @csv' /tmp/aggregated_metrics.json)] line \"Max Open\" [$(jq -r '.db_max_open | @csv' /tmp/aggregated_metrics.json)]" >> $GITHUB_STEP_SUMMARY
echo '```' >> $GITHUB_STEP_SUMMARY
echo "<div align=\"center\">" >> "$GITHUB_STEP_SUMMARY"
echo "" >> "$GITHUB_STEP_SUMMARY"
echo "| Color | Type     |" >> "$GITHUB_STEP_SUMMARY"
echo "|-------|----------|" >> "$GITHUB_STEP_SUMMARY"
echo "| Red   | Max open |" >> "$GITHUB_STEP_SUMMARY"
echo "| Blue  | Idle     |" >> "$GITHUB_STEP_SUMMARY"
echo "| Green | In use   |" >> "$GITHUB_STEP_SUMMARY"
echo "</div>" >> "$GITHUB_STEP_SUMMARY"
echo "" >> "$GITHUB_STEP_SUMMARY"