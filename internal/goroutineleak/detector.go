package goroutineleak

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Detector monitors goroutines and logs their details periodically
type Detector struct {
	logger        *slog.Logger
	interval      time.Duration
	baselineCount int
	previousCount int
	mu            sync.RWMutex
	cancel        context.CancelFunc
}

// Config holds configuration for the goroutine monitor
type Config struct {
	// Interval between snapshots (default: 5 minutes)
	Interval time.Duration
}

// DefaultConfig returns reasonable defaults for monitoring
func DefaultConfig() Config {
	return Config{
		Interval: 5 * time.Minute,
	}
}

// NewDetector creates a new goroutine monitor
func NewDetector(logger *slog.Logger, config Config) *Detector {
	if config.Interval == 0 {
		config.Interval = 5 * time.Minute
	}

	return &Detector{
		logger:        logger,
		interval:      config.Interval,
		baselineCount: runtime.NumGoroutine(),
		previousCount: runtime.NumGoroutine(),
	}
}

// Start begins monitoring goroutines - logs immediately on startup and then periodically
func (d *Detector) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	d.logger.Info("Starting goroutine monitor",
		"baseline", d.baselineCount,
		"interval", d.interval)

	// Log initial state immediately
	d.logGoroutineSnapshot("STARTUP")

	go d.monitor(ctx)
}

// Stop halts the monitor
func (d *Detector) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
}

// GetCurrentCount returns the current goroutine count
func (d *Detector) GetCurrentCount() int {
	return runtime.NumGoroutine()
}

// GetBaseline returns the baseline goroutine count
func (d *Detector) GetBaseline() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.baselineCount
}

func (d *Detector) monitor(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Stopping goroutine monitor")
			return
		case <-ticker.C:
			d.logGoroutineSnapshot("PERIODIC")
		}
	}
}

func (d *Detector) logGoroutineSnapshot(snapshotType string) {
	current := runtime.NumGoroutine()

	d.mu.Lock()
	previous := d.previousCount
	baseline := d.baselineCount
	growth := current - previous
	totalGrowth := current - baseline
	d.previousCount = current
	d.mu.Unlock()

	d.logger.Info("=== GOROUTINE SNAPSHOT ===",
		"type", snapshotType,
		"current", current,
		"previous", previous,
		"baseline", baseline,
		"growth_since_last", growth,
		"growth_since_start", totalGrowth)

	// Always dump goroutine details
	d.dumpGoroutineStacks(current, baseline, totalGrowth)
}

func (d *Detector) dumpGoroutineStacks(current, baseline, totalGrowth int) {
	// Get buffer size estimate
	bufferSize := runtime.NumGoroutine() * 1024
	if bufferSize > 10*1024*1024 {
		bufferSize = 10 * 1024 * 1024 // Cap at 10MB
	}

	buf := make([]byte, bufferSize)
	n := runtime.Stack(buf, true)

	d.logger.Info("Analyzing goroutine stack traces...",
		"stack_dump_bytes", n)

	// Analyze and categorize goroutines
	d.analyzeGoroutines(string(buf[:n]), current, baseline, totalGrowth)

	// Log full stack trace (truncated to prevent log overflow)
	stackStr := string(buf[:n])
	if len(stackStr) > 50000 {
		d.logger.Info("Full stack trace dump (truncated)",
			"total_length", len(stackStr),
			"showing_first", 50000,
			"stacks", stackStr[:50000])
	} else {
		d.logger.Info("Full stack trace dump",
			"stacks", stackStr)
	}
	d.logger.Info("=== END OF SNAPSHOT ===")
}

func (d *Detector) analyzeGoroutines(stackTrace string, current, baseline, totalGrowth int) {
	// Common patterns to categorize goroutines
	patterns := map[string]string{
		"context.Background":              "context.Background without timeout",
		"time.Sleep":                      "in Sleep",
		"chan receive":                    "blocked on channel receive",
		"chan send":                       "blocked on channel send",
		"sync.(*WaitGroup).Wait":          "waiting on WaitGroup",
		"sync.(*Mutex).Lock":              "waiting for mutex",
		"io.ReadFull":                     "blocked on I/O",
		"net/http.(*Client).Do":           "blocked on HTTP request",
		"(*Client).Get":                   "Kubernetes API Get",
		"(*Client).Create":                "Kubernetes API Create",
		"(*Client).Update":                "Kubernetes API Update",
		"internal/process/steps":          "in process steps",
		"internal/btpmanager/credentials": "in BTP credentials",
		"internal/process/provisioning":   "in provisioning",
	}

	// Parse goroutines
	goroutines := strings.Split(stackTrace, "\ngoroutine ")
	categorized := make(map[string][]string)

	for i, goroutine := range goroutines {
		if i == 0 || len(goroutine) == 0 {
			continue
		}

		goroutine = "goroutine " + goroutine
		lines := strings.Split(goroutine, "\n")

		if len(lines) < 3 {
			continue
		}

		header := lines[0]

		// Categorize by pattern
		for pattern, category := range patterns {
			if strings.Contains(goroutine, pattern) {
				// Extract first 8 lines for context
				context := ""
				for j := 0; j < min(8, len(lines)); j++ {
					context += lines[j] + "\n"
				}

				categorized[category] = append(categorized[category],
					fmt.Sprintf("%s\n%s", header, context))
				break
			}
		}
	}

	// Log summary
	d.logger.Info("Goroutine analysis",
		"total", current,
		"categories_found", len(categorized),
		"growth_since_start", totalGrowth)

	// Log each category
	for category, goroutinesList := range categorized {
		count := len(goroutinesList)
		d.logger.Info("Goroutine category",
			"category", category,
			"count", count)

		// Show first 2 examples
		for idx, goroutineInfo := range goroutinesList {
			if idx < 2 {
				d.logger.Info("Example",
					"category", category,
					"example_num", idx+1,
					"stack", goroutineInfo)
			}
		}

		if count > 2 {
			d.logger.Info("Additional goroutines in category",
				"category", category,
				"additional", count-2)
		}
	}

	// Warn if significant growth
	if totalGrowth > 50 {
		d.logger.Warn("Significant goroutine growth detected",
			"growth", totalGrowth,
			"current", current,
			"baseline", baseline)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
