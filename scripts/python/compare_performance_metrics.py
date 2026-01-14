#!/usr/bin/env python3

"""
Performance Metrics Leak Analysis

This script analyzes performance metrics collected during long-duration tests
to detect memory leaks, goroutine leaks, and other resource issues.

It compares baseline metrics (collected before tests) with post-test metrics
(collected after tests and cooldown period) to identify concerning trends.
"""

import json
import os
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import List, Tuple


@dataclass
class MetricThresholds:
    """Configurable thresholds for leak detection"""
    memory_growth_percent: float
    goroutine_increase: int
    fd_increase: int
    db_conn_increase: int


@dataclass
class MetricSnapshot:
    """Snapshot of metrics at a point in time"""
    goroutines: float
    open_fds: float
    mem_alloc: float
    mem_heap: float
    db_idle: float
    db_in_use: float


class Colors:
    """ANSI color codes for terminal output"""
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    NC = '\033[0m'  # No Color


def load_metrics(file_path: Path) -> List[dict]:
    """Load metrics from JSONL file"""
    metrics = []
    with open(file_path, 'r') as f:
        for line in f:
            line = line.strip()
            if line:
                metrics.append(json.loads(line))
    return metrics


def calculate_average_metrics(metrics: List[dict], start_idx: int, end_idx: int) -> MetricSnapshot:
    """Calculate average metrics over a range of samples"""
    samples = metrics[start_idx:end_idx]
    
    if not samples:
        return MetricSnapshot(0, 0, 0, 0, 0, 0)
    
    total = MetricSnapshot(0, 0, 0, 0, 0, 0)
    
    for sample in samples:
        total.goroutines += sample.get('goroutines', 0)
        total.open_fds += sample.get('open_fds', 0)
        total.mem_alloc += sample.get('mem_alloc', 0)
        total.mem_heap += sample.get('mem_heap', 0)
        total.db_idle += sample.get('db_idle', 0)
        total.db_in_use += sample.get('db_in_use', 0)
    
    count = len(samples)
    return MetricSnapshot(
        total.goroutines / count,
        total.open_fds / count,
        total.mem_alloc / count,
        total.mem_heap / count,
        total.db_idle / count,
        total.db_in_use / count
    )


def get_peak_metrics(metrics: List[dict], start_idx: int, end_idx: int) -> Tuple[float, float, float]:
    """Get peak values during test execution"""
    samples = metrics[start_idx:end_idx]
    
    if not samples:
        return 0, 0, 0
    
    peak_goroutines = max(s.get('goroutines', 0) for s in samples)
    peak_memory = max(s.get('mem_alloc', 0) for s in samples)
    peak_fds = max(s.get('open_fds', 0) for s in samples)
    
    return peak_goroutines, peak_memory, peak_fds


def print_section(title: str):
    """Print a section header"""
    print("\n" + "=" * 41)
    print(title)
    print("=" * 41)


def print_metrics(snapshot: MetricSnapshot, title: str):
    """Print metric snapshot"""
    print_section(title)
    print(f"Goroutines:                 {snapshot.goroutines:.0f}")
    print(f"Open FDs:                   {snapshot.open_fds:.0f}")
    print(f"Memory (Alloc):             {snapshot.mem_alloc:.2f} MiB")
    print(f"Memory (Heap):              {snapshot.mem_heap:.2f} MiB")
    print(f"DB Connections (Idle):      {snapshot.db_idle:.0f}")
    print(f"DB Connections (In Use):    {snapshot.db_in_use:.0f}")


def analyze_leaks(baseline: MetricSnapshot, post_test: MetricSnapshot, 
                  peak_goroutines: float, peak_memory: float, peak_fds: float,
                  thresholds: MetricThresholds) -> bool:
    """
    Analyze metrics for leaks. Returns True if leak detected.
    """
    print_section("LEAK DETECTION ANALYSIS")
    leak_detected = False
    
    # Goroutine Analysis
    goroutine_diff = post_test.goroutines - baseline.goroutines
    print("\n🧵 Goroutine Analysis:")
    print(f"  Baseline:  {baseline.goroutines:.0f}")
    print(f"  Post-test: {post_test.goroutines:.0f}")
    print(f"  Difference: {goroutine_diff:.0f}")
    
    if goroutine_diff > thresholds.goroutine_increase:
        print(f"  {Colors.RED}❌ GOROUTINE LEAK DETECTED!{Colors.NC}")
        print(f"  Post-test goroutines increased by {goroutine_diff:.0f} (threshold: {thresholds.goroutine_increase})")
        leak_detected = True
    else:
        print(f"  {Colors.GREEN}✅ No goroutine leak detected{Colors.NC}")
    
    # Memory Analysis
    memory_diff = post_test.mem_alloc - baseline.mem_alloc
    memory_growth_percent = (memory_diff / baseline.mem_alloc * 100) if baseline.mem_alloc > 0 else 0
    
    print("\n💾 Memory Analysis (Allocated):")
    print(f"  Baseline:  {baseline.mem_alloc:.2f} MiB")
    print(f"  Peak:      {peak_memory:.2f} MiB")
    print(f"  Post-test: {post_test.mem_alloc:.2f} MiB")
    print(f"  Difference: {memory_diff:.2f} MiB ({memory_growth_percent:.1f}%)")
    
    if abs(memory_growth_percent) > thresholds.memory_growth_percent:
        print(f"  {Colors.RED}❌ MEMORY LEAK DETECTED!{Colors.NC}")
        print(f"  Memory growth of {memory_growth_percent:.1f}% exceeds threshold of {thresholds.memory_growth_percent}%")
        leak_detected = True
    else:
        print(f"  {Colors.GREEN}✅ No memory leak detected{Colors.NC}")
    
    # Heap Analysis
    heap_diff = post_test.mem_heap - baseline.mem_heap
    heap_growth_percent = (heap_diff / baseline.mem_heap * 100) if baseline.mem_heap > 0 else 0
    
    print("\n🏔️  Heap Memory Analysis:")
    print(f"  Baseline:  {baseline.mem_heap:.2f} MiB")
    print(f"  Post-test: {post_test.mem_heap:.2f} MiB")
    print(f"  Difference: {heap_diff:.2f} MiB ({heap_growth_percent:.1f}%)")
    
    if abs(heap_growth_percent) > thresholds.memory_growth_percent:
        print(f"  {Colors.YELLOW}⚠️  WARNING: Heap growth of {heap_growth_percent:.1f}% exceeds threshold{Colors.NC}")
    else:
        print(f"  {Colors.GREEN}✅ Heap memory stable{Colors.NC}")
    
    # File Descriptor Analysis
    fd_diff = post_test.open_fds - baseline.open_fds
    print("\n📁 File Descriptor Analysis:")
    print(f"  Baseline:  {baseline.open_fds:.0f}")
    print(f"  Post-test: {post_test.open_fds:.0f}")
    print(f"  Difference: {fd_diff:.0f}")
    
    if fd_diff > thresholds.fd_increase:
        print(f"  {Colors.RED}❌ FILE DESCRIPTOR LEAK DETECTED!{Colors.NC}")
        print(f"  Open FDs increased by {fd_diff:.0f} (threshold: {thresholds.fd_increase})")
        leak_detected = True
    else:
        print(f"  {Colors.GREEN}✅ No file descriptor leak detected{Colors.NC}")
    
    # Database Connection Analysis
    db_diff = post_test.db_in_use - baseline.db_in_use
    print("\n🗄️  Database Connection Analysis:")
    print(f"  Baseline (In Use):  {baseline.db_in_use:.0f}")
    print(f"  Post-test (In Use): {post_test.db_in_use:.0f}")
    print(f"  Difference: {db_diff:.0f}")
    
    if db_diff > thresholds.db_conn_increase:
        print(f"  {Colors.RED}❌ DATABASE CONNECTION LEAK DETECTED!{Colors.NC}")
        print(f"  Active DB connections increased by {db_diff:.0f} (threshold: {thresholds.db_conn_increase})")
        leak_detected = True
    else:
        print(f"  {Colors.GREEN}✅ No database connection leak detected{Colors.NC}")
    
    return leak_detected


def main():
    print_section("Performance Metrics Leak Analysis")
    
    # Load configuration from environment variables
    thresholds = MetricThresholds(
        memory_growth_percent=float(os.getenv('MEMORY_GROWTH_THRESHOLD_PERCENT', '70')),
        goroutine_increase=int(os.getenv('GOROUTINE_INCREASE_THRESHOLD', '50')),
        fd_increase=int(os.getenv('FD_INCREASE_THRESHOLD', '10')),
        db_conn_increase=int(os.getenv('DB_CONN_INCREASE_THRESHOLD', '5'))
    )
    
    # Load metrics
    metrics_file = Path('/tmp/keb_metrics.jsonl')
    
    if not metrics_file.exists():
        print(f"{Colors.RED}❌ ERROR: Metrics file not found at {metrics_file}{Colors.NC}")
        sys.exit(1)
    
    try:
        metrics = load_metrics(metrics_file)
    except Exception as e:
        print(f"{Colors.RED}❌ ERROR: Failed to load metrics: {e}{Colors.NC}")
        sys.exit(1)
    
    total_samples = len(metrics)
    print(f"\n📊 Total metric samples collected: {total_samples}")
    
    if total_samples < 20:
        print(f"{Colors.RED}❌ ERROR: Insufficient metrics data (less than 20 samples){Colors.NC}")
        sys.exit(1)
    
    # Calculate sample ranges
    # Check if baseline sample count is specified (from baseline monitoring period)
    baseline_samples_file = Path('/tmp/baseline_samples_count')
    
    if baseline_samples_file.exists():
        try:
            with open(baseline_samples_file, 'r') as f:
                baseline_end_idx = int(f.read().strip())
            # Use samples just before test execution as baseline
            baseline_count = min(50, baseline_end_idx)
            baseline_count = max(5, baseline_count)
            baseline_start_idx = max(0, baseline_end_idx - baseline_count)
            
            print(f"📈 Baseline samples: {baseline_count} readings before test execution (samples {baseline_start_idx}-{baseline_end_idx})")
        except Exception as e:
            print(f"⚠️  Warning: Could not read baseline marker, using first samples: {e}")
            baseline_count = min(50, total_samples // 3)
            baseline_count = max(5, baseline_count)
            baseline_start_idx = 0
            baseline_end_idx = baseline_count
            print(f"📈 Baseline samples: First {baseline_count} readings")
    else:
        # Fallback to first N samples
        baseline_count = min(50, total_samples // 3)
        baseline_count = max(5, baseline_count)
        baseline_start_idx = 0
        baseline_end_idx = baseline_count
        print(f"📈 Baseline samples: First {baseline_count} readings")
    
    post_test_count = min(50, total_samples // 3)
    post_test_count = max(5, post_test_count)
    
    print(f"📉 Post-test samples: Last {post_test_count} readings")
    
    # Calculate metrics
    baseline = calculate_average_metrics(metrics, baseline_start_idx, baseline_end_idx)
    post_test = calculate_average_metrics(metrics, total_samples - post_test_count, total_samples)
    
    # Calculate peak values during test execution (after baseline, before post-test)
    test_start = baseline_end_idx
    test_end = total_samples - post_test_count
    peak_goroutines, peak_memory, peak_fds = get_peak_metrics(metrics, test_start, test_end)
    
    # Print results
    print_metrics(baseline, "BASELINE METRICS (Before Tests)")
    
    print_section("PEAK METRICS (During Test Execution)")
    print(f"Peak Goroutines:            {peak_goroutines:.0f}")
    print(f"Peak Open FDs:              {peak_fds:.0f}")
    print(f"Peak Memory:                {peak_memory:.2f} MiB")
    
    print_metrics(post_test, "POST-TEST METRICS (After Cooldown)")
    
    # Analyze for leaks
    leak_detected = analyze_leaks(baseline, post_test, peak_goroutines, peak_memory, peak_fds, thresholds)
    
    # Final result
    print_section("FINAL RESULT")
    
    if leak_detected:
        print(f"{Colors.RED}❌ LEAK DETECTION TEST FAILED{Colors.NC}")
        print("\nOne or more resource leaks were detected during the long-duration test.")
        print("Please review the metrics above and investigate the identified issues.")
        print("\nThresholds used:")
        print(f"  - Memory growth: {thresholds.memory_growth_percent}%")
        print(f"  - Goroutine increase: {thresholds.goroutine_increase}")
        print(f"  - File descriptor increase: {thresholds.fd_increase}")
        print(f"  - DB connection increase: {thresholds.db_conn_increase}")
        sys.exit(1)
    else:
        print(f"{Colors.GREEN}✅ LEAK DETECTION TEST PASSED{Colors.NC}")
        print("\nNo significant resource leaks detected.")
        print("All metrics are within acceptable thresholds after cooldown period.")
        sys.exit(0)


if __name__ == '__main__':
    main()
