package api

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

// --- Banned words flow -------------------------------------------------------

func TestCheckBannedWordsDeletesBelowThreshold(t *testing.T) {
	fs := &fakeStore{bannedWords: []database.BannedWord{{WordPattern: "spam"}}}
	d := newTestDiscordWith(fs, NewCache(1000, 3, time.Hour)) // threshold 3
	fake := &fakeSender{}

	m := testMessage()
	m.Content = "this is spam"

	if !d.checkBannedWords(fake, m) {
		t.Fatal("checkBannedWords should return true on a match")
	}
	// First violation (1) is below the threshold (3): delete, do not ban.
	if len(fake.deletes) != 1 {
		t.Errorf("expected 1 delete, got %d", len(fake.deletes))
	}
	if len(fake.bans) != 0 {
		t.Errorf("expected no ban below threshold, got %d", len(fake.bans))
	}
	// The action is recorded in the moderation log.
	if len(fs.modLogEntries) != 1 || fs.modLogEntries[0].action != model.ActionDeleteMessage {
		t.Errorf("expected one DELETE_MESSAGE log entry, got %+v", fs.modLogEntries)
	}
	// The author gets a warning DM plus a copy of their message.
	if len(fake.dms) < 1 {
		t.Error("expected the author to be warned by DM")
	}
}

func TestCheckBannedWordsBansAtThreshold(t *testing.T) {
	fs := &fakeStore{bannedWords: []database.BannedWord{{WordPattern: "spam"}}}
	d := newTestDiscordWith(fs, NewCache(1000, 1, time.Hour)) // threshold 1
	fake := &fakeSender{}

	m := testMessage()
	m.Content = "spam"

	if !d.checkBannedWords(fake, m) {
		t.Fatal("checkBannedWords should return true on a match")
	}
	// First violation (1) reaches the threshold (1): ban, do not delete.
	if len(fake.bans) != 1 {
		t.Errorf("expected 1 ban at threshold, got %d", len(fake.bans))
	}
	if len(fake.deletes) != 0 {
		t.Errorf("expected no delete at threshold, got %d", len(fake.deletes))
	}
}

func TestCheckBannedWordsDisabled(t *testing.T) {
	fs := &fakeStore{
		settings:    &database.AutomoderationSettings{BannedWordsEnabled: false},
		bannedWords: []database.BannedWord{{WordPattern: "spam"}},
	}
	d := newTestDiscordWith(fs, NewCache(1000, 3, time.Hour))
	fake := &fakeSender{}

	m := testMessage()
	m.Content = "spam"

	if d.checkBannedWords(fake, m) {
		t.Error("disabled feature should not match")
	}
	if len(fake.deletes)+len(fake.bans) != 0 {
		t.Error("disabled feature should take no action")
	}
}

func TestCheckBannedWordsNoMatch(t *testing.T) {
	fs := &fakeStore{bannedWords: []database.BannedWord{{WordPattern: "spam"}}}
	d := newTestDiscordWith(fs, NewCache(1000, 3, time.Hour))
	fake := &fakeSender{}

	m := testMessage()
	m.Content = "a perfectly fine message"

	if d.checkBannedWords(fake, m) {
		t.Error("clean message should not match")
	}
	if len(fake.deletes)+len(fake.bans) != 0 {
		t.Error("clean message should trigger no action")
	}
}

// --- Banned websites flow ----------------------------------------------------

func TestCheckBannedWebsitesDeletes(t *testing.T) {
	fs := &fakeStore{bannedWebsites: []database.BannedWebsite{{WebsiteURL: "evil.com"}}}
	d := newTestDiscordWith(fs, NewCache(1000, 3, time.Hour))
	fake := &fakeSender{}

	m := testMessage()
	m.Content = "join https://evil.com/x now"

	if !d.checkBannedWebsites(fake, m) {
		t.Fatal("checkBannedWebsites should return true on a match")
	}
	if len(fake.deletes) != 1 {
		t.Errorf("expected 1 delete, got %d", len(fake.deletes))
	}
}

func TestCheckBannedWebsitesNoURLInMessage(t *testing.T) {
	fs := &fakeStore{bannedWebsites: []database.BannedWebsite{{WebsiteURL: "evil.com"}}}
	d := newTestDiscordWith(fs, NewCache(1000, 3, time.Hour))
	fake := &fakeSender{}

	m := testMessage()
	m.Content = "just talking, no links here"

	if d.checkBannedWebsites(fake, m) {
		t.Error("message without a URL should not match")
	}
	if len(fake.deletes) != 0 {
		t.Error("no action expected when there is no URL")
	}
}

// --- Spam detection flow -----------------------------------------------------

func TestCheckMessageSpamBansCrossChannelDuplicates(t *testing.T) {
	fs := &fakeStore{}
	c := NewCache(1000, 2, time.Hour) // threshold 2
	d := newTestDiscordWith(fs, c)
	fake := &fakeSender{}

	// Same content posted by u1 in two different channels, already in cache.
	c.AddMessage(msg("g1", "c1", "m1", "u1", "buy now"))
	c.AddMessage(msg("g1", "c2", "m2", "u1", "buy now"))

	// A third identical message in a third channel trips the cross-channel rule.
	m := &discordgo.Message{
		ID: "m3", GuildID: "g1", ChannelID: "c3", Content: "buy now",
		Author: &discordgo.User{ID: "u1", Username: "spammer"},
	}

	if !d.checkMessageSpam(fake, m) {
		t.Fatal("checkMessageSpam should detect cross-channel spam")
	}
	if len(fake.bans) != 1 {
		t.Errorf("expected 1 ban for spam, got %d", len(fake.bans))
	}
}

func TestCheckMessageSpamIgnoresSingleChannel(t *testing.T) {
	fs := &fakeStore{}
	c := NewCache(1000, 2, time.Hour)
	d := newTestDiscordWith(fs, c)
	fake := &fakeSender{}

	// Two duplicates but all in the SAME channel: not cross-channel spam.
	c.AddMessage(msg("g1", "c1", "m1", "u1", "hi"))
	c.AddMessage(msg("g1", "c1", "m2", "u1", "hi"))

	m := &discordgo.Message{
		ID: "m3", GuildID: "g1", ChannelID: "c1", Content: "hi",
		Author: &discordgo.User{ID: "u1", Username: "u"},
	}

	if d.checkMessageSpam(fake, m) {
		t.Error("same-channel duplicates should not be treated as spam")
	}
	if len(fake.bans) != 0 {
		t.Error("no ban expected for single-channel repetition")
	}
}

// --- moderateMessage orchestration ------------------------------------------

func TestModerateMessageSkipsModerators(t *testing.T) {
	fs := &fakeStore{
		isModerator: true,
		bannedWords: []database.BannedWord{{WordPattern: "spam"}},
	}
	d := newTestDiscordWith(fs, NewCache(1000, 1, time.Hour))
	fake := &fakeSender{}

	m := testMessage()
	m.Content = "spam"
	m.Member = &discordgo.Member{}

	d.moderateMessage(fake, m)

	if len(fake.bans)+len(fake.deletes) != 0 {
		t.Error("messages from moderators must be ignored entirely")
	}
}

func TestModerateMessageSkipsExcludedChannel(t *testing.T) {
	fs := &fakeStore{
		excluded:    map[string]bool{"c1": true},
		bannedWords: []database.BannedWord{{WordPattern: "spam"}},
	}
	d := newTestDiscordWith(fs, NewCache(1000, 1, time.Hour))
	fake := &fakeSender{}

	m := testMessage()
	m.Content = "spam"
	m.ChannelID = "c1"
	m.Member = &discordgo.Member{}

	d.moderateMessage(fake, m)

	if len(fake.bans)+len(fake.deletes) != 0 {
		t.Error("messages in excluded channels must not be moderated")
	}
}

func TestModerateMessageActsOnViolation(t *testing.T) {
	fs := &fakeStore{bannedWords: []database.BannedWord{{WordPattern: "spam"}}}
	d := newTestDiscordWith(fs, NewCache(1000, 1, time.Hour)) // threshold 1 -> ban
	fake := &fakeSender{}

	m := testMessage()
	m.Content = "spam"
	m.Member = &discordgo.Member{}

	d.moderateMessage(fake, m)

	if len(fake.bans) != 1 {
		t.Errorf("expected the banned word to lead to a ban, got %d bans", len(fake.bans))
	}
}
