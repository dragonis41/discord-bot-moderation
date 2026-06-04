package api

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

// msg builds a minimal MessageCreate for cache tests.
func msg(guildID, channelID, msgID, authorID, content string) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        msgID,
			GuildID:   guildID,
			ChannelID: channelID,
			Content:   content,
			Author:    &discordgo.User{ID: authorID, Username: "user-" + authorID},
		},
	}
}

func TestCacheAddAndGetUserRecentMessages(t *testing.T) {
	c := NewCache(100, 3, time.Hour)

	c.AddMessage(msg("g1", "c1", "m1", "u1", "hello"))
	c.AddMessage(msg("g1", "c1", "m2", "u2", "other user"))
	c.AddMessage(msg("g1", "c2", "m3", "u1", "world"))

	got := c.GetUserRecentMessages("g1", "u1", 10)
	if len(got) != 2 {
		t.Fatalf("GetUserRecentMessages(u1) = %d messages, want 2", len(got))
	}
	// Most recent first.
	if got[0].Content != "world" || got[1].Content != "hello" {
		t.Errorf("unexpected order: %q then %q", got[0].Content, got[1].Content)
	}

	// The limit is respected.
	if n := len(c.GetUserRecentMessages("g1", "u1", 1)); n != 1 {
		t.Errorf("limit=1 returned %d messages", n)
	}

	// Unknown guild/user yields nothing.
	if got := c.GetUserRecentMessages("nope", "u1", 10); got != nil {
		t.Errorf("unknown guild = %v, want nil", got)
	}
}

func TestCacheEvictsBeyondMaxSize(t *testing.T) {
	c := NewCache(2, 3, time.Hour)

	c.AddMessage(msg("g1", "c1", "m1", "u1", "first"))
	c.AddMessage(msg("g1", "c1", "m2", "u1", "second"))
	c.AddMessage(msg("g1", "c1", "m3", "u1", "third")) // evicts m1

	got := c.GetGuildRecentMessages("g1", 10)
	if len(got) != 2 {
		t.Fatalf("cache held %d messages, want max 2", len(got))
	}
	// Newest first; oldest ("first") evicted.
	if got[0].Content != "third" || got[1].Content != "second" {
		t.Errorf("unexpected retained messages: %q, %q", got[0].Content, got[1].Content)
	}
}

func TestCacheUpdateMessage(t *testing.T) {
	c := NewCache(100, 3, time.Hour)
	c.AddMessage(msg("g1", "c1", "m1", "u1", "before"))

	c.UpdateMessage(&discordgo.MessageUpdate{
		Message: &discordgo.Message{
			ID:        "m1",
			GuildID:   "g1",
			ChannelID: "c1",
			Content:   "after",
			Author:    &discordgo.User{ID: "u1"},
		},
	})

	got := c.GetUserRecentMessages("g1", "u1", 1)
	if len(got) != 1 || got[0].Content != "after" {
		t.Fatalf("update not applied: %+v", got)
	}

	// Updating an unknown message is a no-op (must not panic or insert).
	c.UpdateMessage(&discordgo.MessageUpdate{
		Message: &discordgo.Message{ID: "ghost", GuildID: "g1", Author: &discordgo.User{ID: "u1"}},
	})
	if n := len(c.GetGuildRecentMessages("g1", 10)); n != 1 {
		t.Errorf("unknown update changed cache size to %d", n)
	}
}

func TestCacheViolationIncrements(t *testing.T) {
	c := NewCache(100, 3, time.Hour)

	if n := c.IncrementViolation("g1", "u1"); n != 1 {
		t.Errorf("first violation = %d, want 1", n)
	}
	if n := c.IncrementViolation("g1", "u1"); n != 2 {
		t.Errorf("second violation = %d, want 2", n)
	}
	if n := c.GetViolationCount("g1", "u1"); n != 2 {
		t.Errorf("GetViolationCount = %d, want 2", n)
	}

	// Different user tracked separately.
	if n := c.IncrementViolation("g1", "u2"); n != 1 {
		t.Errorf("other user first violation = %d, want 1", n)
	}

	c.ResetViolations("g1", "u1")
	if n := c.GetViolationCount("g1", "u1"); n != 0 {
		t.Errorf("after reset = %d, want 0", n)
	}
}

func TestCacheViolationWindowExpiry(t *testing.T) {
	// A window in the past forces every violation to be treated as fresh and any
	// existing count to read as expired — exercising the window-reset branches
	// deterministically without sleeping.
	c := NewCache(100, 3, -time.Second)

	if n := c.IncrementViolation("g1", "u1"); n != 1 {
		t.Errorf("first = %d, want 1", n)
	}
	// Second increment sees an expired window and resets to 1 instead of 2.
	if n := c.IncrementViolation("g1", "u1"); n != 1 {
		t.Errorf("after expired window = %d, want 1 (reset)", n)
	}
	// GetViolationCount also treats the expired window as zero.
	if n := c.GetViolationCount("g1", "u1"); n != 0 {
		t.Errorf("GetViolationCount with expired window = %d, want 0", n)
	}
}

func TestCacheConfigGetters(t *testing.T) {
	c := NewCache(42, 7, 3*time.Minute)
	if c.GetMaxCacheSize() != 42 {
		t.Errorf("GetMaxCacheSize = %d, want 42", c.GetMaxCacheSize())
	}
	if c.GetViolationThreshold() != 7 {
		t.Errorf("GetViolationThreshold = %d, want 7", c.GetViolationThreshold())
	}
	if c.GetViolationWindow() != 3*time.Minute {
		t.Errorf("GetViolationWindow = %v, want 3m", c.GetViolationWindow())
	}
}

func TestStickerAndAttachmentHelpers(t *testing.T) {
	items := []*discordgo.StickerItem{{ID: "s1", Name: "a"}, {ID: "s2", Name: "b"}}
	if got := stickerIDsFromItems(items); got != "s1,s2" {
		t.Errorf("stickerIDsFromItems = %q, want s1,s2", got)
	}

	cached := cachedStickers(items)
	if len(cached) != 2 || cached[0].ID != "s1" || cached[1].Name != "b" {
		t.Errorf("cachedStickers = %+v", cached)
	}
	if got := stickerIDsFromCache(cached); got != "s1,s2" {
		t.Errorf("stickerIDsFromCache = %q, want s1,s2", got)
	}

	// The two encodings must agree, since spam detection compares them.
	if stickerIDsFromCache(cached) != stickerIDsFromItems(items) {
		t.Error("cache and item sticker encodings disagree")
	}

	atts := []*discordgo.MessageAttachment{{Filename: "x.png"}, {Filename: "y.gif"}}
	names := attachmentNames(atts)
	if len(names) != 2 || names[0] != "x.png" || names[1] != "y.gif" {
		t.Errorf("attachmentNames = %v", names)
	}
}

func TestCachePendingBan(t *testing.T) {
	c := NewCache(100, 3, time.Hour)

	if c.IsUserPendingBan("g1", "u1") {
		t.Error("user should not be pending before marking")
	}
	if !c.MarkUserAsPendingBan("g1", "u1") {
		t.Error("first mark should return true")
	}
	if c.MarkUserAsPendingBan("g1", "u1") {
		t.Error("second mark within window should return false")
	}
	if !c.IsUserPendingBan("g1", "u1") {
		t.Error("user should be pending after marking")
	}

	c.ClearPendingBan("g1", "u1")
	if c.IsUserPendingBan("g1", "u1") {
		t.Error("user should not be pending after clear")
	}
	// After clearing, marking succeeds again.
	if !c.MarkUserAsPendingBan("g1", "u1") {
		t.Error("mark after clear should return true")
	}
}
