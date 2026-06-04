package utils

import (
	"testing"
	"time"
)

func TestGetUptimeUnknownBeforeInit(t *testing.T) {
	// uptime is a package-level var; reset it so the test does not depend on
	// whether NewUptime ran earlier in the binary.
	uptime = nil
	if got := GetUptime(); got != "Unknown" {
		t.Errorf("GetUptime() before init = %q, want Unknown", got)
	}
}

func TestGetUptimeAfterInit(t *testing.T) {
	NewUptime()
	if got := GetUptime(); got == "Unknown" || got == "" {
		t.Errorf("GetUptime() after init = %q, want a duration string", got)
	}

	d := uptime.GetUptime()
	if d < 0 || d > time.Hour {
		t.Errorf("uptime duration %v is implausible for a fresh start", d)
	}
}
