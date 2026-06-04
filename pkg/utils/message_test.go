package utils

import (
	"strings"
	"testing"
)

func TestSplitMessageShortStaysWhole(t *testing.T) {
	out := SplitMessage("hello", 100)
	if len(out) != 1 || out[0] != "hello" {
		t.Fatalf("SplitMessage = %v, want [hello]", out)
	}
}

func TestSplitMessageReconstructs(t *testing.T) {
	inputs := []string{
		"hello world this is a sentence with several words and punctuation.",
		strings.Repeat("a", 1000),
		strings.Repeat("word ", 300),
		"oneverylongtokenwithoutanyspacesatallthatmustbehardsplit" + strings.Repeat("x", 500),
	}
	limits := []int{5, 10, 50, 200}

	for _, in := range inputs {
		for _, limit := range limits {
			parts := SplitMessage(in, limit)

			// Joining the parts must reproduce the original exactly.
			if got := strings.Join(parts, ""); got != in {
				t.Errorf("limit=%d: reconstruction mismatch", limit)
			}
			// No part may exceed the limit (measured in runes, as the impl does).
			for i, p := range parts {
				if n := len([]rune(p)); n > limit {
					t.Errorf("limit=%d: part %d has %d runes (> limit)", limit, i, n)
				}
			}
		}
	}
}

func TestSplitMessagePrefersWhitespace(t *testing.T) {
	// "hello world" with limit 8 should break on the space rather than mid-word.
	parts := SplitMessage("hello world", 8)
	if len(parts) == 0 || !strings.HasSuffix(parts[0], " ") {
		t.Fatalf("expected first part to end at the space, got %q", parts[0])
	}
}

func TestSplitMessageHandlesMultibyteRunes(t *testing.T) {
	// Each emoji is multiple bytes; splitting must not corrupt them.
	in := strings.Repeat("🚨", 20)
	parts := SplitMessage(in, 5)
	if got := strings.Join(parts, ""); got != in {
		t.Fatalf("multibyte reconstruction mismatch")
	}
	for _, p := range parts {
		if !utf8Valid(p) {
			t.Fatalf("split produced invalid UTF-8: %q", p)
		}
	}
}

func utf8Valid(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}
