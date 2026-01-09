#!/usr/bin/env bash

# This script analyzes performance metrics collected during long-duration tests
# to detect memory leaks, goroutine leaks, and other resource issues.
#
# It compares baseline metrics (collected before tests) with post-test metrics
# (collected after tests and cooldown period) to identify concerning trends.

set -e

# Configuration - Thresholds for leak detection
MEMORY_GROWTH_THRESHOLD_PERCENT=${MEMORY_GROWTH_THRESHOLD_PERCENT:-70}    # Max acceptable memory growth % (can be overridden by env)
GOROUTINE_INCREASE_THRESHOLD=${GOROUTINE_INCREASE_THRESHOLD:-50}       # Max acceptable goroutine increase (can be overridden by env)
FD_INCREASE_THRESHOLD=${FD_INCREASE_THRESHOLD:-10}              # Max acceptable file descriptor increase (can be overridden by env)
DB_CONN_INCREASE_THRESHOLD=${DB_CONN_INCREASE_THRESHOLD:-5}          # Max acceptable DB connection increase (can be overridden by env)

# Metric file
METRICS_FILE="/tmp/keb_metrics.jsonl"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "========================================="
echo "Performance Metrics Leak Analysis"
echo "========================================="
echo ""

# Check if metrics file exists
if [ ! -f "$METRICS_FILE" ]; then
    echo -e "${RED}❌ ERROR: Metrics file not found at $METRICS_FILE${NC}"
    exit 1
fi

# Count total metrics collected
TOTAL_LINES=$(wc -l < "$METRICS_FILE")
echo "📊 Total metric samples collected: $TOTAL_LINES"

if [ "$TOTAL_LINES" -lt 20 ]; then
    echo -e "${RED}❌ ERROR: Insufficient metrics data (less than 20 samples)${NC}"
    exit 1
fi

# Calculate baseline and post-test ranges
# Baseline: First N samples
# Post-test: Last N samples
BASELINE_COUNT=50
POST_TEST_COUNT=50

# Ensure we don't exceed available samples
if [ "$BASELINE_COUNT" -gt "$((TOTAL_LINES / 3))" ]; then
    BASELINE_COUNT=$((TOTAL_LINES / 3))
fi

if [ "$POST_TEST_COUNT" -gt "$((TOTAL_LINES / 3))" ]; then
    POST_TEST_COUNT=$((TOTAL_LINES / 3))
fi

# Minimum of 5 samples required
if [ "$BASELINE_COUNT" -lt 5 ]; then
    BASELINE_COUNT=5
fi

if [ "$POST_TEST_COUNT" -lt 5 ]; then
    POST_TEST_COUNT=5
fi

echo "📈 Baseline samples: First $BASELINE_COUNT readings"
echo "📉 Post-test samples: Last $POST_TEST_COUNT readings"
echo ""

# Function to calculate average of a metric from a range of lines
calculate_average() {
    local metric_name=$1
    local start_line=$2
    local end_line=$3
    
    # Map metric names to JSON keys used by monitor_metrics.sh
    local json_key=""
    case "$metric_name" in
        "go_goroutines") json_key="goroutines" ;;
        "process_open_fds") json_key="open_fds" ;;
        "go_memstats_alloc_bytes_mib") json_key="mem_alloc" ;;
        "go_memstats_heap_inuse_bytes_mib") json_key="mem_heap" ;;
        "go_sql_stats_connections_idle") json_key="db_idle" ;;
        "go_sql_stats_connections_in_use") json_key="db_in_use" ;;
        *) json_key="$metric_name" ;;
    esac
    
    # Extract metric values from specified line range
    sed -n "${start_line},${end_line}p" "$METRICS_FILE" | while read -r line; do
        value=$(echo "$line" | jq -r ".${json_key} // 0")
        if [ "$value" != "null" ] && [ "$value" != "0" ]; then
            echo "$value"
        fi
    done | awk '{sum+=$1; count++} END {if(count>0) print sum/count; else print 0}'
}

# Function to get max value of a metric from a range
get_max_value() {
    local metric_name=$1
    local start_line=$2
    local end_line=$3
    
    # Map metric names to JSON keys used by monitor_metrics.sh
    local json_key=""
    case "$metric_name" in
        "go_goroutines") json_key="goroutines" ;;
        "process_open_fds") json_key="open_fds" ;;
        "go_memstats_alloc_bytes_mib") json_key="mem_alloc" ;;
        "go_memstats_heap_inuse_bytes_mib") json_key="mem_heap" ;;
        "go_sql_stats_connections_idle") json_key="db_idle" ;;
        "go_sql_stats_connections_in_use") json_key="db_in_use" ;;
        *) json_key="$metric_name" ;;
    esac
    
    sed -n "${start_line},${end_line}p" "$METRICS_FILE" | while read -r line; do
        value=$(echo "$line" | jq -r ".${json_key} // 0")
        if [ "$value" != "null" ]; then
            echo "$value"
        fi
    done | sort -n | tail -1
}

# Calculate baseline metrics (average of first N samples)
echo "🔍 Calculating baseline metrics..."
BASELINE_END=$BASELINE_COUNT
BASELINE_GOROUTINES=$(calculate_average "go_goroutines" 1 "$BASELINE_END")
BASELINE_FDS=$(calculate_average "process_open_fds" 1 "$BASELINE_END")
BASELINE_MEMORY=$(calculate_average "go_memstats_alloc_bytes_mib" 1 "$BASELINE_END")
BASELINE_HEAP=$(calculate_average "go_memstats_heap_inuse_bytes_mib" 1 "$BASELINE_END")
BASELINE_DB_IDLE=$(calculate_average "go_sql_stats_connections_idle" 1 "$BASELINE_END")
BASELINE_DB_INUSE=$(calculate_average "go_sql_stats_connections_in_use" 1 "$BASELINE_END")

# Calculate post-test metrics (average of last N samples)
echo "🔍 Calculating post-test metrics..."
POST_TEST_START=$((TOTAL_LINES - POST_TEST_COUNT + 1))
POST_TEST_GOROUTINES=$(calculate_average "go_goroutines" "$POST_TEST_START" "$TOTAL_LINES")
POST_TEST_FDS=$(calculate_average "process_open_fds" "$POST_TEST_START" "$TOTAL_LINES")
POST_TEST_MEMORY=$(calculate_average "go_memstats_alloc_bytes_mib" "$POST_TEST_START" "$TOTAL_LINES")
POST_TEST_HEAP=$(calculate_average "go_memstats_heap_inuse_bytes_mib" "$POST_TEST_START" "$TOTAL_LINES")
POST_TEST_DB_IDLE=$(calculate_average "go_sql_stats_connections_idle" "$POST_TEST_START" "$TOTAL_LINES")
POST_TEST_DB_INUSE=$(calculate_average "go_sql_stats_connections_in_use" "$POST_TEST_START" "$TOTAL_LINES")

# Also get peak values during test execution (middle portion)
echo "🔍 Calculating peak values during test execution..."
TEST_START=$((BASELINE_COUNT + 1))
TEST_END=$((TOTAL_LINES - POST_TEST_COUNT))
PEAK_GOROUTINES=$(get_max_value "go_goroutines" "$TEST_START" "$TEST_END")
PEAK_MEMORY=$(get_max_value "go_memstats_alloc_bytes_mib" "$TEST_START" "$TEST_END")
PEAK_FDS=$(get_max_value "process_open_fds" "$TEST_START" "$TEST_END")

echo ""
echo "========================================="
echo "BASELINE METRICS (Before Tests)"
echo "========================================="
printf "Goroutines:        %.0f\n" "$BASELINE_GOROUTINES"
printf "Open FDs:          %.0f\n" "$BASELINE_FDS"
printf "Memory (Alloc):    %.2f MiB\n" "$BASELINE_MEMORY"
printf "Memory (Heap):     %.2f MiB\n" "$BASELINE_HEAP"
printf "DB Connections (Idle):   %.0f\n" "$BASELINE_DB_IDLE"
printf "DB Connections (In Use): %.0f\n" "$BASELINE_DB_INUSE"

echo ""
echo "========================================="
echo "PEAK METRICS (During Test Execution)"
echo "========================================="
printf "Peak Goroutines:   %.0f\n" "$PEAK_GOROUTINES"
printf "Peak Open FDs:     %.0f\n" "$PEAK_FDS"
printf "Peak Memory:       %.2f MiB\n" "$PEAK_MEMORY"

echo ""
echo "========================================="
echo "POST-TEST METRICS (After Cooldown)"
echo "========================================="
printf "Goroutines:        %.0f\n" "$POST_TEST_GOROUTINES"
printf "Open FDs:          %.0f\n" "$POST_TEST_FDS"
printf "Memory (Alloc):    %.2f MiB\n" "$POST_TEST_MEMORY"
printf "Memory (Heap):     %.2f MiB\n" "$POST_TEST_HEAP"
printf "DB Connections (Idle):   %.0f\n" "$POST_TEST_DB_IDLE"
printf "DB Connections (In Use): %.0f\n" "$POST_TEST_DB_INUSE"

echo ""
echo "========================================="
echo "LEAK DETECTION ANALYSIS"
echo "========================================="

# Track if any leak detected
LEAK_DETECTED=0

# Analyze Goroutines
GOROUTINE_DIFF=$(awk "BEGIN {printf \"%.0f\", $POST_TEST_GOROUTINES - $BASELINE_GOROUTINES}")
echo ""
echo "🧵 Goroutine Analysis:"
printf "  Baseline: %.0f\n" "$BASELINE_GOROUTINES"
printf "  Post-test: %.0f\n" "$POST_TEST_GOROUTINES"
printf "  Difference: %d\n" "$GOROUTINE_DIFF"

if [ "$GOROUTINE_DIFF" -gt "$GOROUTINE_INCREASE_THRESHOLD" ]; then
    echo -e "  ${RED}❌ GOROUTINE LEAK DETECTED!${NC}"
    echo "  Post-test goroutines increased by $GOROUTINE_DIFF (threshold: $GOROUTINE_INCREASE_THRESHOLD)"
    LEAK_DETECTED=1
else
    echo -e "  ${GREEN}✅ No goroutine leak detected${NC}"
fi

# Analyze Memory
MEMORY_DIFF=$(awk "BEGIN {printf \"%.2f\", $POST_TEST_MEMORY - $BASELINE_MEMORY}")
MEMORY_GROWTH_PERCENT=$(awk "BEGIN {if($BASELINE_MEMORY>0) printf \"%.1f\", ($POST_TEST_MEMORY - $BASELINE_MEMORY) / $BASELINE_MEMORY * 100; else print 0}")
echo ""
echo "💾 Memory Analysis (Allocated):"
printf "  Baseline: %.2f MiB\n" "$BASELINE_MEMORY"
printf "  Peak: %.2f MiB\n" "$PEAK_MEMORY"
printf "  Post-test: %.2f MiB\n" "$POST_TEST_MEMORY"
printf "  Difference: %.2f MiB (%.1f%%)\n" "$MEMORY_DIFF" "$MEMORY_GROWTH_PERCENT"

MEMORY_GROWTH_ABS=$(awk "BEGIN {
    val = $MEMORY_GROWTH_PERCENT;
    print (val < 0) ? -val : val
}")
MEMORY_THRESHOLD_CHECK=$(awk "BEGIN {print ($MEMORY_GROWTH_ABS > $MEMORY_GROWTH_THRESHOLD_PERCENT) ? 1 : 0}")

if [ "$MEMORY_THRESHOLD_CHECK" -eq 1 ]; then
    echo -e "  ${RED}❌ MEMORY LEAK DETECTED!${NC}"
    echo "  Memory growth of ${MEMORY_GROWTH_PERCENT}% exceeds threshold of ${MEMORY_GROWTH_THRESHOLD_PERCENT}%"
    LEAK_DETECTED=1
else
    echo -e "  ${GREEN}✅ No memory leak detected${NC}"
fi

# Analyze Heap Memory
HEAP_DIFF=$(awk "BEGIN {printf \"%.2f\", $POST_TEST_HEAP - $BASELINE_HEAP}")
HEAP_GROWTH_PERCENT=$(awk "BEGIN {if($BASELINE_HEAP>0) printf \"%.1f\", ($POST_TEST_HEAP - $BASELINE_HEAP) / $BASELINE_HEAP * 100; else print 0}")
echo ""
echo "🏔️  Heap Memory Analysis:"
printf "  Baseline: %.2f MiB\n" "$BASELINE_HEAP"
printf "  Post-test: %.2f MiB\n" "$POST_TEST_HEAP"
printf "  Difference: %.2f MiB (%.1f%%)\n" "$HEAP_DIFF" "$HEAP_GROWTH_PERCENT"

HEAP_GROWTH_ABS=$(awk "BEGIN {
    val = $HEAP_GROWTH_PERCENT;
    print (val < 0) ? -val : val
}")
HEAP_THRESHOLD_CHECK=$(awk "BEGIN {print ($HEAP_GROWTH_ABS > $MEMORY_GROWTH_THRESHOLD_PERCENT) ? 1 : 0}")

if [ "$HEAP_THRESHOLD_CHECK" -eq 1 ]; then
    echo -e "  ${YELLOW}⚠️  WARNING: Heap growth of ${HEAP_GROWTH_PERCENT}% exceeds threshold${NC}"
else
    echo -e "  ${GREEN}✅ Heap memory stable${NC}"
fi

# Analyze File Descriptors
FD_DIFF=$(awk "BEGIN {printf \"%.0f\", $POST_TEST_FDS - $BASELINE_FDS}")
echo ""
echo "📁 File Descriptor Analysis:"
printf "  Baseline: %.0f\n" "$BASELINE_FDS"
printf "  Post-test: %.0f\n" "$POST_TEST_FDS"
printf "  Difference: %d\n" "$FD_DIFF"

if [ "$FD_DIFF" -gt "$FD_INCREASE_THRESHOLD" ]; then
    echo -e "  ${RED}❌ FILE DESCRIPTOR LEAK DETECTED!${NC}"
    echo "  Open FDs increased by $FD_DIFF (threshold: $FD_INCREASE_THRESHOLD)"
    LEAK_DETECTED=1
else
    echo -e "  ${GREEN}✅ No file descriptor leak detected${NC}"
fi

# Analyze Database Connections
DB_INUSE_DIFF=$(awk "BEGIN {printf \"%.0f\", $POST_TEST_DB_INUSE - $BASELINE_DB_INUSE}")
echo ""
echo "🗄️  Database Connection Analysis:"
printf "  Baseline (In Use): %.0f\n" "$BASELINE_DB_INUSE"
printf "  Post-test (In Use): %.0f\n" "$POST_TEST_DB_INUSE"
printf "  Difference: %d\n" "$DB_INUSE_DIFF"

if [ "$DB_INUSE_DIFF" -gt "$DB_CONN_INCREASE_THRESHOLD" ]; then
    echo -e "  ${RED}❌ DATABASE CONNECTION LEAK DETECTED!${NC}"
    echo "  Active DB connections increased by $DB_INUSE_DIFF (threshold: $DB_CONN_INCREASE_THRESHOLD)"
    LEAK_DETECTED=1
else
    echo -e "  ${GREEN}✅ No database connection leak detected${NC}"
fi

echo ""
echo "========================================="
echo "FINAL RESULT"
echo "========================================="

if [ "$LEAK_DETECTED" -eq 1 ]; then
    echo -e "${RED}❌ LEAK DETECTION TEST FAILED${NC}"
    echo ""
    echo "One or more resource leaks were detected during the long-duration test."
    echo "Please review the metrics above and investigate the identified issues."
    echo ""
    echo "Thresholds used:"
    echo "  - Memory growth: ${MEMORY_GROWTH_THRESHOLD_PERCENT}%"
    echo "  - Goroutine increase: ${GOROUTINE_INCREASE_THRESHOLD}"
    echo "  - File descriptor increase: ${FD_INCREASE_THRESHOLD}"
    echo "  - DB connection increase: ${DB_CONN_INCREASE_THRESHOLD}"
    exit 1
else
    echo -e "${GREEN}✅ LEAK DETECTION TEST PASSED${NC}"
    echo ""
    echo "No significant resource leaks detected."
    echo "All metrics are within acceptable thresholds after cooldown period."
    exit 0
fi