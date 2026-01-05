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

// Detector monitors goroutine counts to detect potential leaks
type Detector struct {
	logger              *slog.Logger
	interval            time.Duration
	growthThreshold     int
	baselineCount       int
	previousCount       int
	consecutiveGrowth   int
	maxConsecutiveGrows int
	mu                  sync.RWMutex
	cancel              context.CancelFunc
}

// Config holds configuration for the goroutine leak detector
type Config struct {
	// Interval between checks
	Interval time.Duration
	// GrowthThreshold is the minimum number of goroutines that must increase to trigger detection
	GrowthThreshold int
	// MaxConsecutiveGrows is how many consecutive increases before alerting
	MaxConsecutiveGrows int
}

// DefaultConfig returns reasonable defaults for leak detection
func DefaultConfig() Config {
	return Config{
		Interval:            30 * time.Second,
		GrowthThreshold:     50,
		MaxConsecutiveGrows: 3,
	}
}

// NewDetector creates a new goroutine leak detector
func NewDetector(logger *slog.Logger, config Config) *Detector {
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}
	if config.GrowthThreshold == 0 {
		config.GrowthThreshold = 50
	}
	if config.MaxConsecutiveGrows == 0 {
		config.MaxConsecutiveGrows = 3
	}

	return &Detector{
		logger:              logger,
		interval:            config.Interval,
		growthThreshold:     config.GrowthThreshold,
		maxConsecutiveGrows: config.MaxConsecutiveGrows,
		baselineCount:       runtime.NumGoroutine(),
		previousCount:       runtime.NumGoroutine(),
	}
}

// Start begins monitoring goroutines
func (d *Detector) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	d.logger.Info("Starting goroutine leak detector",
		"baseline", d.baselineCount,
		"interval", d.interval,
		"threshold", d.growthThreshold)

	go d.monitor(ctx)
}

// Stop halts the detector
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

// ResetBaseline updates the baseline to the current count
func (d *Detector) ResetBaseline() {
	d.mu.Lock()
	defer d.mu.Unlock()

	current := runtime.NumGoroutine()
	d.baselineCount = current
	d.previousCount = current
	d.consecutiveGrowth = 0

	d.logger.Info("Reset goroutine baseline", "count", current)
}

func (d *Detector) monitor(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Stopping goroutine leak detector")
			return
		case <-ticker.C:
			d.check()
		}
	}
}

func (d *Detector) check() {
	current := runtime.NumGoroutine()

	d.mu.Lock()
	previous := d.previousCount
	baseline := d.baselineCount
	growth := current - previous
	totalGrowth := current - baseline
	d.previousCount = current

	// Check for growth
	if growth >= d.growthThreshold {
		d.consecutiveGrowth++
		d.logger.Warn("Goroutine count increased significantly",
			"current", current,
			"previous", previous,
			"growth", growth,
			"baseline", baseline,
			"total_growth", totalGrowth,
			"consecutive_grows", d.consecutiveGrowth)

		if d.consecutiveGrowth >= d.maxConsecutiveGrows {
			d.logger.Error("POTENTIAL GOROUTINE LEAK DETECTED",
				"current", current,
				"baseline", baseline,
				"total_growth", totalGrowth,
				"consecutive_grows", d.consecutiveGrowth,
				"threshold", d.growthThreshold)

			// Log goroutine stack traces for debugging
			d.dumpGoroutineStacks()
		}
	} else if growth < -d.growthThreshold {
		// Significant decrease, reset consecutive counter
		d.consecutiveGrowth = 0
		d.logger.Info("Goroutine count decreased",
			"current", current,
			"previous", previous,
			"decrease", -growth)
	} else {
		// Stable or minor change
		if d.consecutiveGrowth > 0 {
			d.consecutiveGrowth = 0
			d.logger.Info("Goroutine count stabilized", "current", current)
		}
	}
	d.mu.Unlock()

	// Log periodic status
	if current > baseline+100 {
		d.logger.Info("Goroutine monitoring status",
			"current", current,
			"baseline", baseline,
			"growth_from_baseline", totalGrowth)
	}
}

func (d *Detector) dumpGoroutineStacks() {
	// Get buffer size estimate
	bufferSize := runtime.NumGoroutine() * 1024
	if bufferSize > 10*1024*1024 {
		bufferSize = 10 * 1024 * 1024 // Cap at 10MB
	}

	buf := make([]byte, bufferSize)
	n := runtime.Stack(buf, true)

	d.logger.Error("=== GOROUTINE LEAK DETECTED - ANALYZING STACK TRACES ===",
		"total_goroutines", runtime.NumGoroutine(),
		"stack_dump_bytes", n)

	// Analyze and identify potential leak sources
	d.identifyLeakSources(string(buf[:n]))

	// Log full stack trace (truncated to prevent log overflow)
	stackStr := string(buf[:n])
	if len(stackStr) > 50000 {
		d.logger.Error("Full stack trace dump (truncated)",
			"total_length", len(stackStr),
			"showing_first", 50000,
			"stacks", stackStr[:50000])
	} else {
		d.logger.Error("Full stack trace dump",
			"stacks", stackStr)
	}
	d.logger.Error("=== END OF STACK TRACES ===")
}

func (d *Detector) identifyLeakSources(stackTrace string) {
	d.logger.Info("Analyzing stack traces for leak sources...")

	// Common patterns that indicate leaks
	leakPatterns := map[string]string{
		"context.Background":              "Using context.Background without timeout",
		"time.Sleep":                      "Goroutine in long sleep",
		"chan receive":                    "Blocked on channel receive",
		"chan send":                       "Blocked on channel send",
		"sync.(*WaitGroup).Wait":          "Waiting on WaitGroup",
		"sync.(*Mutex).Lock":              "Waiting for mutex lock",
		"io.ReadFull":                     "Blocked on I/O read",
		"net/http.(*Client).Do":           "Blocked on HTTP request",
		"(*Client).Get":                   "Blocked on Kubernetes API call",
		"(*Client).Create":                "Blocked on Kubernetes API call",
		"(*Client).Update":                "Blocked on Kubernetes API call",
		"internal/process/steps":          "In provisioning step",
		"internal/btpmanager/credentials": "In BTP credentials manager",
		"internal/process/provisioning":   "In provisioning process",
	}

	// Parse goroutines
	goroutines := strings.Split(stackTrace, "\ngoroutine ")
	suspiciousGoroutines := make(map[string][]string)

	for i, goroutine := range goroutines {
		if i == 0 || len(goroutine) == 0 {
			continue
		}

		goroutine = "goroutine " + goroutine
		lines := strings.Split(goroutine, "\n")

		if len(lines) < 3 {
			continue
		}

		header := lines[0] // e.g., "goroutine 123 [chan receive]:"

		// Check for leak patterns
		for pattern, description := range leakPatterns {
			if strings.Contains(goroutine, pattern) {
				// Extract relevant stack frames (first 6 lines usually show the issue)
				context := ""
				for j := 0; j < min(8, len(lines)); j++ {
					context += lines[j] + "\n"
				}

				key := description
				suspiciousGoroutines[key] = append(suspiciousGoroutines[key],
					fmt.Sprintf("%s\n%s", header, context))
				break
			}
		}
	}

	if len(suspiciousGoroutines) > 0 {
		d.logger.Warn("Found potentially leaking goroutines",
			"categories", len(suspiciousGoroutines))

		for category, goroutines := range suspiciousGoroutines {
			count := len(goroutines)
			d.logger.Warn("\n╔═══════════════════════════════════════════════════════════")
			d.logger.Warn("║ LEAK CATEGORY", "type", category)
			d.logger.Warn("║ COUNT", "goroutines", count)
			d.logger.Warn("╚═══════════════════════════════════════════════════════════")

			// Show first 3 examples of each category
			for idx, goroutineInfo := range goroutines {
				if idx < 3 {
					d.logger.Warn("Example",
						"number", idx+1,
						"stack", goroutineInfo)
				}
			}

			if count > 3 {
				d.logger.Warn("More goroutines in this category",
					"additional_count", count-3)
			}
		}
	} else {
		d.logger.Info("No obvious leak patterns detected in stack traces")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
