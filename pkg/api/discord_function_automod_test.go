package api

import (
	"testing"

	"github.com/dragonis41/discord-bot-moderation/internal/database"
)

func TestBannedWordMatchesLiteral(t *testing.T) {
	w := database.BannedWord{WordPattern: "spam", IsRegex: false}

	tests := []struct {
		content string
		want    bool
	}{
		{"this is spam right here", true}, // surrounded by spaces
		{"spam", true},                    // whole message
		{"SPAM in caps", true},            // case-insensitive
		{"stop, spam!", true},             // bounded by punctuation
		{"spamming is different", false},  // substring, not a whole word
		{"notspam", false},                // substring at end
		{"clean message", false},          // absent
		{"", false},                       // empty
	}
	for _, tt := range tests {
		got, err := bannedWordMatches(tt.content, w)
		if err != nil {
			t.Fatalf("bannedWordMatches(%q) error: %v", tt.content, err)
		}
		if got != tt.want {
			t.Errorf("bannedWordMatches(%q) = %v, want %v", tt.content, got, tt.want)
		}
	}
}

func TestBannedWordMatchesRegex(t *testing.T) {
	w := database.BannedWord{WordPattern: "ba+d", IsRegex: true}

	for _, c := range []string{"that is baaad", "bad word"} {
		got, err := bannedWordMatches(c, w)
		if err != nil {
			t.Fatalf("bannedWordMatches(%q) error: %v", c, err)
		}
		if !got {
			t.Errorf("regex pattern should match %q", c)
		}
	}

	if got, _ := bannedWordMatches("good", w); got {
		t.Error("regex pattern should not match 'good'")
	}
}

func TestBannedWordMatchesLiteralDotIsEscaped(t *testing.T) {
	// A literal (non-regex) word with a dot must be treated literally, so the dot
	// only matches a real dot, not any character.
	w := database.BannedWord{WordPattern: "a.b", IsRegex: false}

	if got, _ := bannedWordMatches("a.b", w); !got {
		t.Error("literal 'a.b' should match 'a.b'")
	}
	if got, _ := bannedWordMatches("axb", w); got {
		t.Error("literal 'a.b' should NOT match 'axb' (dot must be escaped)")
	}
}

func TestBannedWordMatchesInvalidRegexReturnsError(t *testing.T) {
	w := database.BannedWord{WordPattern: "(unclosed", IsRegex: true}
	if _, err := bannedWordMatches("anything", w); err == nil {
		t.Error("invalid regex should return an error")
	}
}

func TestBannedWebsiteMatches(t *testing.T) {
	w := database.BannedWebsite{WebsiteURL: "evil.com"}

	tests := []struct {
		content string
		want    bool
	}{
		{"check out https://evil.com/page", true},
		{"visit EVIL.COM now", true}, // case-insensitive
		{"a good link to safe.org", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := bannedWebsiteMatches(tt.content, w); got != tt.want {
			t.Errorf("bannedWebsiteMatches(%q) = %v, want %v", tt.content, got, tt.want)
		}
	}
}

func TestContainsURL(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"https://example.com", true},
		{"go to example.com please", true},
		{"www.example.org", true},
		{"just some plain text with no link", false},
		{"a sentence. ending normally", false}, // period + space is not a domain
		{"the word.word token looks like a domain", true},
	}
	// The "word.word" case documents a known property: the URL regex matches bare
	// "word.word" sequences with no space, so a token can look like a domain even
	// in prose. That is why checkBannedWebsites still verifies against the actual
	// banned list rather than treating any URL-like token as a violation.
	for _, tt := range tests {
		if got := containsURL(tt.content); got != tt.want {
			t.Errorf("containsURL(%q) = %v, want %v", tt.content, got, tt.want)
		}
	}
}
