package api

import (
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
)

// --- Banned word commands ----------------------------------------------------

func TestAddBannedWordCommand(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.addBannedWord(fake, modInteraction("add-banned-word", strOpt("word", "badword"), boolOpt("is_regex", true)))

	if len(fs.addedWords) != 1 {
		t.Fatalf("expected 1 added word, got %d", len(fs.addedWords))
	}
	if fs.addedWords[0].pattern != "badword" || !fs.addedWords[0].isRegex {
		t.Errorf("added word = %+v, want {badword true}", fs.addedWords[0])
	}
	if len(fake.followups) != 1 {
		t.Errorf("expected 1 confirmation follow-up, got %d", len(fake.followups))
	}
}

func TestAddBannedWordCommandRejectsEmpty(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.addBannedWord(fake, modInteraction("add-banned-word", strOpt("word", "")))

	if len(fs.addedWords) != 0 {
		t.Errorf("empty word should not be added, got %d", len(fs.addedWords))
	}
	if len(fake.followups) != 1 {
		t.Errorf("expected an error follow-up, got %d", len(fake.followups))
	}
}

func TestAddBannedWordCommandPermissionDenied(t *testing.T) {
	fs := &fakeStore{denyPermission: true}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.addBannedWord(fake, modInteraction("add-banned-word", strOpt("word", "x")))

	if len(fs.addedWords) != 0 {
		t.Error("a non-moderator must not be able to add a banned word")
	}
}

func TestRemoveBannedWordCommand(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.removeBannedWord(fake, modInteraction("remove-banned-word", intOpt("id", 7)))

	if len(fs.removedWords) != 1 || fs.removedWords[0] != 7 {
		t.Errorf("removedWords = %v, want [7]", fs.removedWords)
	}
}

func TestRemoveBannedWordCommandRejectsZeroID(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.removeBannedWord(fake, modInteraction("remove-banned-word", intOpt("id", 0)))

	if len(fs.removedWords) != 0 {
		t.Error("id 0 should be rejected")
	}
}

func TestListBannedWordsCommand(t *testing.T) {
	fs := &fakeStore{bannedWords: []database.BannedWord{{ID: 1, WordPattern: "spam"}}}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.listBannedWords(fake, modInteraction("get-banned-words"))

	if len(fake.followups) != 1 {
		t.Fatalf("expected 1 follow-up, got %d", len(fake.followups))
	}
	desc := fake.followups[0].Embeds[0].Description
	if !strings.Contains(desc, "spam") {
		t.Errorf("listing should contain the banned word, got %q", desc)
	}
}

func TestListBannedWordsCommandEmpty(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.listBannedWords(fake, modInteraction("get-banned-words"))

	if len(fake.followups) != 1 {
		t.Fatalf("expected 1 follow-up, got %d", len(fake.followups))
	}
	if !strings.Contains(fake.followups[0].Embeds[0].Description, "Aucun") {
		t.Error("empty listing should show the 'none configured' message")
	}
}

// --- Banned website commands -------------------------------------------------

func TestAddBannedWebsiteCommand(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.addBannedWebsite(fake, modInteraction("add-banned-website", strOpt("url", "evil.com")))

	if len(fs.addedWebsites) != 1 || fs.addedWebsites[0] != "evil.com" {
		t.Errorf("addedWebsites = %v, want [evil.com]", fs.addedWebsites)
	}
}

func TestRemoveBannedWebsiteCommand(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.removeBannedWebsite(fake, modInteraction("remove-banned-website", intOpt("id", 3)))

	if len(fs.removedWebsites) != 1 || fs.removedWebsites[0] != 3 {
		t.Errorf("removedWebsites = %v, want [3]", fs.removedWebsites)
	}
}

func TestListBannedWebsitesCommand(t *testing.T) {
	fs := &fakeStore{bannedWebsites: []database.BannedWebsite{{ID: 1, WebsiteURL: "evil.com"}}}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.listBannedWebsites(fake, modInteraction("get-banned-websites"))

	if len(fake.followups) != 1 {
		t.Fatalf("expected 1 follow-up, got %d", len(fake.followups))
	}
	if !strings.Contains(fake.followups[0].Embeds[0].Description, "evil.com") {
		t.Error("listing should contain the banned website")
	}
}

func TestConfigureAutomodCommand(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.configureAutomod(fake, modInteraction("configure-automod"))

	// A fresh interaction (no existing Message) posts the selection menu as a
	// new follow-up carrying components.
	if len(fake.followups) != 1 {
		t.Fatalf("expected the selection menu follow-up, got %d", len(fake.followups))
	}
	if len(fake.followups[0].Components) == 0 {
		t.Error("selection menu follow-up should carry components")
	}
}

// --- Log commands ------------------------------------------------------------

func TestGetBotLogsCommand(t *testing.T) {
	fs := &fakeStore{
		sysLogCount:   5,
		sysLogMax:     100,
		sysLogEntries: []database.SystemLogEntry{{Function: "fn", Content: "an event", CreatedAt: "2024-01-01 00:00:00"}},
	}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.getBotLogs(fake, modInteraction("get-bot-logs"))

	if len(fake.followups) != 1 {
		t.Fatalf("expected 1 follow-up, got %d", len(fake.followups))
	}
	if len(fake.followups[0].Embeds) == 0 {
		t.Error("bot logs follow-up should carry an embed")
	}
}

func TestGetModerationLogsCommand(t *testing.T) {
	fs := &fakeStore{
		modLogCount: 2,
		modLogMax:   100,
		modLogList:  []database.ModerationLogEntry{{Action: "BAN", UserID: "u1", Username: "bad", Reason: "spam", CreatedAt: "2024-01-01 00:00:00"}},
	}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.getModerationLogs(fake, modInteraction("get-moderation-logs"))

	if len(fake.followups) != 1 {
		t.Fatalf("expected 1 follow-up, got %d", len(fake.followups))
	}
}

// --- Selection-menu commands -------------------------------------------------

func TestSelectLogChannelsCommand(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{
		guildChannels: []*discordgo.Channel{
			{ID: "c1", Name: "general", Type: discordgo.ChannelTypeGuildText},
			{ID: "v1", Name: "voice", Type: discordgo.ChannelTypeGuildVoice}, // filtered out
		},
	}

	d.selectLogChannels(fake, modInteraction("set-log-channels"))

	if len(fake.followups) != 1 {
		t.Fatalf("expected the selection menu follow-up, got %d", len(fake.followups))
	}
	if len(fake.followups[0].Components) == 0 {
		t.Error("selection menu should carry components")
	}
}

func TestSelectModeratorRolesCommand(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{
		guildRoles: []*discordgo.Role{{ID: "r1", Name: "Mods", Position: 1}},
	}

	d.selectModeratorRoles(fake, modInteraction("set-moderation-roles"))

	if len(fake.followups) != 1 {
		t.Fatalf("expected the selection menu follow-up, got %d", len(fake.followups))
	}
}
