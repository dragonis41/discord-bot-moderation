package utils

import (
	"strings"
	"testing"
)

func TestGetMemoryUsage(t *testing.T) {
	out := GetMemoryUsage()
	for _, want := range []string{"Alloc", "Sys", "GC"} {
		if !strings.Contains(out, want) {
			t.Errorf("GetMemoryUsage() = %q, missing %q", out, want)
		}
	}
}

func TestGetDetailedStats(t *testing.T) {
	stats := GetDetailedStats()
	for _, key := range []string{"Memory Allocated", "Goroutines", "GC Runs"} {
		if _, ok := stats[key]; !ok {
			t.Errorf("GetDetailedStats() missing key %q", key)
		}
	}
}

func TestGetCPUUsageNotInitialized(t *testing.T) {
	systemStats = nil // reset the package-level state
	if got := GetCPUUsage(); !strings.Contains(got, "not initialized") {
		t.Errorf("GetCPUUsage() before init = %q, want 'not initialized'", got)
	}
}

func TestInitSystemStatsThenCPUUsage(t *testing.T) {
	if err := InitSystemStats(); err != nil {
		t.Fatalf("InitSystemStats: %v", err)
	}
	got := GetCPUUsage()
	if strings.Contains(got, "not initialized") {
		t.Errorf("after InitSystemStats, GetCPUUsage = %q, want a percentage", got)
	}
	if !strings.HasSuffix(got, "%") {
		t.Errorf("GetCPUUsage = %q, want a value ending in %%", got)
	}
}
