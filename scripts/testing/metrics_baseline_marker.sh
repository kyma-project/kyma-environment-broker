#!/usr/bin/env bash
# Mark the end of baseline period for metrics
# Usage: metrics_baseline_marker.sh

set -o nounset
set -o errexit
set -E
set -o pipefail

wc -l < /tmp/keb_metrics.jsonl > /tmp/baseline_samples_count
echo "Baseline marker set at $(cat /tmp/baseline_samples_count) samples"
