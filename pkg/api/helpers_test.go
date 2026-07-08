package api

import (
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
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
	e := errorEmbed(i18n.EN, "Title", "Desc")
	if e.Title != "Title" || e.Description != "Desc" {
		t.Errorf("errorEmbed fields = %q/%q", e.Title, e.Description)
	}
	if e.Color != model.Red.Int() {
		t.Errorf("errorEmbed color = %#x, want red", e.Color)
	}
	if successEmbed(i18n.EN, "a", "b").Color != model.Green.Int() {
		t.Error("successEmbed should be green")
	}
	if infoEmbed(i18n.EN, "a", "b").Color != model.Blue.Int() {
		t.Error("infoEmbed should be blue")
	}
	// All builders attach the localized hint footer and a timestamp.
	if e.Footer == nil || e.Footer.Text != i18n.T(i18n.EN, "footer.hint") {
		t.Error("errorEmbed should use the localized hint footer")
	}
	if e.Timestamp == "" {
		t.Error("errorEmbed should set a timestamp")
	}
}

// --- Standalone helpers ------------------------------------------------------

func TestGuildNameByID(t *testing.T) {
	fake := &fakeSender{}
	if got := guildNameByID(fake, "g1"); got != "TestGuild" {
		t.Errorf("guildNameByID = %q, want TestGuild", got)
	}
}

func TestFormatDiscordTimestamp(t *testing.T) {
	// A parseable timestamp becomes Discord's <t:unix:f> form.
	got := formatDiscordTimestamp("2024-01-02 03:04:05")
	if !strings.HasPrefix(got, "<t:") || !strings.HasSuffix(got, ":f>") {
		t.Errorf("formatted timestamp = %q, want <t:...:f>", got)
	}

	// An unparseable string is returned unchanged.
	if got := formatDiscordTimestamp("not a date"); got != "not a date" {
		t.Errorf("unparseable timestamp = %q, want it returned verbatim", got)
	}
}

func TestBuildReportActionButtons(t *testing.T) {
	components := buildReportActionButtons("u123")
	if len(components) != 1 {
		t.Fatalf("expected 1 action row, got %d", len(components))
	}
	row, ok := components[0].(discordgo.ActionsRow)
	if !ok {
		t.Fatalf("expected an ActionsRow, got %T", components[0])
	}
	if len(row.Components) != 3 {
		t.Fatalf("expected 3 buttons (kick/ban/ignore), got %d", len(row.Components))
	}

	wantIDs := map[string]bool{
		"report_kick_u123":   false,
		"report_ban_u123":    false,
		"report_ignore_u123": false,
	}
	for _, comp := range row.Components {
		btn := comp.(discordgo.Button)
		if _, ok := wantIDs[btn.CustomID]; !ok {
			t.Errorf("unexpected button CustomID %q", btn.CustomID)
		}
		wantIDs[btn.CustomID] = true
	}
	for id, seen := range wantIDs {
		if !seen {
			t.Errorf("missing button %q", id)
		}
	}
}

func TestLogHelpersPersistToStore(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(1000, 3, time.Hour))

	d.logError("g1", "fn", "an error %d", 1)
	d.logInfo("g1", "fn", "info")
	d.logWarning("g1", "fn", "warn")
	d.logSuccess("g1", "fn", "ok")

	if fs.systemLogs != 4 {
		t.Errorf("expected 4 persisted system log entries, got %d", fs.systemLogs)
	}
}
