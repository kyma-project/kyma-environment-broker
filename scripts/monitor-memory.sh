#!/usr/bin/env bash
# Monitor KEB pod memory usage and record peak.
# Run in a separate terminal while perf-test.sh is running.
# Usage: ./scripts/monitor-memory.sh [interval_seconds]
# Output: prints live usage, writes CSV to /tmp/keb-memory-<timestamp>.csv

set -o nounset
set -o pipefail

INTERVAL=${1:-2}
TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
CSV="/tmp/keb-memory-${TIMESTAMP}.csv"
PEAK_MEM=0
PEAK_CPU=0

echo "time,cpu_cores,memory_mi" > "$CSV"
echo "Monitoring KEB memory (interval=${INTERVAL}s) — Ctrl+C to stop"
echo "Output: $CSV"
echo ""
printf "%-20s %10s %12s %12s\n" "TIME" "CPU(cores)" "MEMORY(Mi)" "PEAK(Mi)"
printf "%-20s %10s %12s %12s\n" "----" "----------" "----------" "--------"

while true; do
  LINE=$(kubectl top pod -n kcp-system -l app.kubernetes.io/name=kyma-environment-broker \
    --no-headers 2>/dev/null | head -1)

  if [[ -z "$LINE" ]]; then
    sleep "$INTERVAL"
    continue
  fi

  CPU=$(echo "$LINE" | awk '{print $2}' | tr -d 'm')
  MEM=$(echo "$LINE" | awk '{print $3}' | tr -d 'Mi')
  NOW=$(date '+%H:%M:%S')

  if [[ "$MEM" -gt "$PEAK_MEM" ]]; then
    PEAK_MEM=$MEM
  fi
  if [[ "$CPU" -gt "$PEAK_CPU" ]]; then
    PEAK_CPU=$CPU
  fi

  printf "%-20s %10s %12s %12s\n" "$NOW" "${CPU}m" "${MEM}Mi" "${PEAK_MEM}Mi"
  echo "$(date '+%H:%M:%S'),$CPU,$MEM" >> "$CSV"

  sleep "$INTERVAL"
done
