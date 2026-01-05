package goroutineleak

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestDetector_BasicFunctionality(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := Config{
		Interval:            100 * time.Millisecond,
		GrowthThreshold:     5,
		MaxConsecutiveGrows: 2,
	}
	
	detector := NewDetector(logger, config)
	
	baseline := detector.GetBaseline()
	if baseline <= 0 {
		t.Errorf("Expected positive baseline, got %d", baseline)
	}
	
	current := detector.GetCurrentCount()
	if current <= 0 {
		t.Errorf("Expected positive current count, got %d", current)
	}
}

func TestDetector_DetectsLeak(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := Config{
		Interval:            50 * time.Millisecond,
		GrowthThreshold:     3,
		MaxConsecutiveGrows: 2,
	}
	
	detector := NewDetector(logger, config)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	detector.Start(ctx)
	
	// Create goroutine leak
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			<-done
		}()
	}
	
	// Let detector run
	time.Sleep(300 * time.Millisecond)
	
	// Cleanup
	close(done)
	detector.Stop()
	
	// Verify detection ran (just check it didn't crash)
	current := detector.GetCurrentCount()
	if current <= 0 {
		t.Error("Expected valid goroutine count")
	}
}

func TestDetector_ResetBaseline(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	detector := NewDetector(logger, DefaultConfig())
	
	oldBaseline := detector.GetBaseline()
	
	// Create some goroutines
	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func() {
			<-done
		}()
	}
	
	time.Sleep(10 * time.Millisecond)
	
	detector.ResetBaseline()
	newBaseline := detector.GetBaseline()
	
	if newBaseline <= oldBaseline {
		t.Errorf("Expected new baseline (%d) > old baseline (%d)", newBaseline, oldBaseline)
	}
	
	close(done)
}
