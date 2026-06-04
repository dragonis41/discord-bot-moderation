package utils

import (
	"os"
	"testing"
)

// TestCheckEnvironmentVariablesHappyPath exercises the non-fatal branch: when the
// token is present the function returns normally. The missing-token branch calls
// LogFatal (os.Exit) and is therefore not unit-tested here.
func TestCheckEnvironmentVariablesHappyPath(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "dummy-token")

	if os.Getenv("DISCORD_BOT_TOKEN") == "" {
		t.Fatal("test setup failed: token not set")
	}

	// Should not call LogFatal / exit.
	CheckEnvironmentVariables()
}
