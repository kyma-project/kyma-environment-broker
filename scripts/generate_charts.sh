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

# Check if metrics file exists and has data
if [ ! -f "$METRICS_FILE" ]; then
  echo "❌ ERROR: Metrics file not found at $METRICS_FILE" >> $GITHUB_STEP_SUMMARY
  exit 1
fi

# Check if file has content
if [ ! -s "$METRICS_FILE" ]; then
  echo "❌ ERROR: Metrics file is empty" >> $GITHUB_STEP_SUMMARY
  exit 1
fi

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

# Verify we have data points
DATA_POINTS=$(jq '.goroutines | length' /tmp/aggregated_metrics.json)
if [ "$DATA_POINTS" -eq 0 ]; then
  echo "❌ ERROR: No metric data points collected" >> $GITHUB_STEP_SUMMARY
  exit 1
fi

echo "📊 Collected $DATA_POINTS metric samples" >> $GITHUB_STEP_SUMMARY
echo "" >> $GITHUB_STEP_SUMMARY

# Calculate analysis ranges
if [ -f "/tmp/baseline_samples_count" ]; then
  BASELINE_END=$(cat /tmp/baseline_samples_count)
  BASELINE_COUNT=$(( BASELINE_END < 50 ? BASELINE_END : 50 ))
  BASELINE_COUNT=$(( BASELINE_COUNT < 5 ? 5 : BASELINE_COUNT ))
  BASELINE_START=$(( BASELINE_END - BASELINE_COUNT ))
  BASELINE_START=$(( BASELINE_START < 0 ? 0 : BASELINE_START ))
else
  BASELINE_COUNT=$(( DATA_POINTS / 3 ))
  BASELINE_COUNT=$(( BASELINE_COUNT > 50 ? 50 : BASELINE_COUNT ))
  BASELINE_COUNT=$(( BASELINE_COUNT < 5 ? 5 : BASELINE_COUNT ))
  BASELINE_START=0
  BASELINE_END=$BASELINE_COUNT
fi

POST_TEST_COUNT=$(( DATA_POINTS / 3 ))
POST_TEST_COUNT=$(( POST_TEST_COUNT > 50 ? 50 : POST_TEST_COUNT ))
POST_TEST_COUNT=$(( POST_TEST_COUNT < 5 ? 5 : POST_TEST_COUNT ))
POST_TEST_START=$(( DATA_POINTS - POST_TEST_COUNT ))

echo "### 📊 Metrics Timeline Overview" >> $GITHUB_STEP_SUMMARY
echo "" >> $GITHUB_STEP_SUMMARY
echo '```' >> $GITHUB_STEP_SUMMARY
echo "Total samples: ${DATA_POINTS}" >> $GITHUB_STEP_SUMMARY
echo "" >> $GITHUB_STEP_SUMMARY
echo "Timeline:" >> $GITHUB_STEP_SUMMARY
echo "├─ 🔵 Baseline Period:  samples ${BASELINE_START}-${BASELINE_END} (${BASELINE_COUNT} samples)" >> $GITHUB_STEP_SUMMARY
echo "├─ ⚡ Test Execution:   samples ${BASELINE_END}-${POST_TEST_START} ($((POST_TEST_START - BASELINE_END)) samples)" >> $GITHUB_STEP_SUMMARY
echo "└─ 🔴 Post-Test Period: samples ${POST_TEST_START}-${DATA_POINTS} (${POST_TEST_COUNT} samples)" >> $GITHUB_STEP_SUMMARY
echo "" >> $GITHUB_STEP_SUMMARY
echo "Leak analysis compares 🔵 Baseline vs 🔴 Post-Test metrics" >> $GITHUB_STEP_SUMMARY
echo '```' >> $GITHUB_STEP_SUMMARY
echo "" >> $GITHUB_STEP_SUMMARY
      
echo '```mermaid' >> $GITHUB_STEP_SUMMARY
echo "xychart-beta title \"Goroutines\" line \"Goroutines\" [$(jq -r '.goroutines | @csv' /tmp/aggregated_metrics.json)]" >> $GITHUB_STEP_SUMMARY
echo '```' >> $GITHUB_STEP_SUMMARY

echo '```mermaid' >> $GITHUB_STEP_SUMMARY
echo "xychart-beta title \"Open FDs\" line \"open_fds\" [$(jq -r '.open_fds | @csv' /tmp/aggregated_metrics.json)]" >> $GITHUB_STEP_SUMMARY
echo '```' >> $GITHUB_STEP_SUMMARY

# For scheduled/manual runs, split memory charts to avoid Mermaid text size limit
if [[ "$GITHUB_EVENT_NAME" == "schedule" || "$GITHUB_EVENT_NAME" == "workflow_dispatch" ]]; then
  echo '```mermaid' >> $GITHUB_STEP_SUMMARY
  echo "xychart-beta title \"Memory - Alloc\" y-axis \"Memory (in MiB)\" line \"Alloc\" [$(jq -r '.mem_alloc | @csv' /tmp/aggregated_metrics.json)]" >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
  
  echo '```mermaid' >> $GITHUB_STEP_SUMMARY
  echo "xychart-beta title \"Memory - Heap\" y-axis \"Memory (in MiB)\" line \"Heap\" [$(jq -r '.mem_heap | @csv' /tmp/aggregated_metrics.json)]" >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
  
  echo '```mermaid' >> $GITHUB_STEP_SUMMARY
  echo "xychart-beta title \"Memory - Stack\" y-axis \"Memory (in MiB)\" line \"Stack\" [$(jq -r '.mem_stack | @csv' /tmp/aggregated_metrics.json)]" >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
else
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
fi

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