package api

import (
	"strings"
	"testing"

	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},         // shorter than max: unchanged
		{"hello", 5, "hello"},          // exactly max: unchanged
		{"hello world", 8, "hello..."}, // longer: cut to max-3 + "..."
		{"abcdef", 4, "a..."},          // cut to 1 char + "..."
	}
	for _, tt := range tests {
		if got := truncate(tt.in, tt.max); got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
		}
		if len(truncate(tt.in, tt.max)) > max(tt.max, len(tt.in)) {
			t.Errorf("truncate(%q, %d) exceeded max length", tt.in, tt.max)
		}
	}
}

func TestSanitizeInlineCode(t *testing.T) {
	// Backticks must be neutralized so they cannot break out of an inline span.
	if strings.Contains(sanitizeInlineCode("a`b`c"), "`") {
		t.Error("sanitizeInlineCode left a real backtick in the output")
	}
	// Newlines and carriage returns become spaces (can't live in inline code).
	got := sanitizeInlineCode("line1\nline2\rline3")
	if strings.ContainsAny(got, "\n\r") {
		t.Errorf("sanitizeInlineCode left a line break: %q", got)
	}
	// Plain text is untouched.
	if got := sanitizeInlineCode("plain text"); got != "plain text" {
		t.Errorf("sanitizeInlineCode(plain) = %q, want unchanged", got)
	}
}

func TestJoinLinesWithLimit(t *testing.T) {
	lines := []string{"aaa\n", "bbb\n", "ccc\n"}

	// Generous limit keeps everything.
	full := joinLinesWithLimit(lines, 1000)
	if full != "aaa\nbbb\nccc\n" {
		t.Errorf("joinLinesWithLimit(full) = %q", full)
	}

	// Tight limit stops early and appends the truncation notice.
	short := joinLinesWithLimit(lines, 5)
	if !strings.Contains(short, "logs truncated") {
		t.Errorf("expected truncation notice, got %q", short)
	}
	if strings.Contains(short, "ccc") {
		t.Errorf("expected later lines to be dropped, got %q", short)
	}
}

func TestEmbedBuilders(t *testing.T) {
	e := errorEmbed("Title", "Desc")
	if e.Title != "Title" || e.Description != "Desc" {
		t.Errorf("errorEmbed fields = %q/%q", e.Title, e.Description)
	}
	if e.Color != model.Red.Int() {
		t.Errorf("errorEmbed color = %#x, want red", e.Color)
	}
	if successEmbed("a", "b").Color != model.Green.Int() {
		t.Error("successEmbed should be green")
	}
	if infoEmbed("a", "b").Color != model.Blue.Int() {
		t.Error("infoEmbed should be blue")
	}
	// All builders attach the default footer and a timestamp.
	if e.Footer != model.DefaultFooter {
		t.Error("errorEmbed should use the default footer")
	}
	if e.Timestamp == "" {
		t.Error("errorEmbed should set a timestamp")
	}
}
