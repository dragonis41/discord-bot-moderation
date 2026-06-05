package api

import (
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
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

func TestRemoveBannedWordCommandShowsMenu(t *testing.T) {
	fs := &fakeStore{bannedWords: []database.BannedWord{{ID: 7, WordPattern: "spam"}}}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.removeBannedWord(fake, modInteraction("remove-banned-word"))

	if len(fake.followups) != 1 {
		t.Fatalf("expected 1 follow-up, got %d", len(fake.followups))
	}
	if len(fake.followups[0].Components) == 0 {
		t.Error("the removal menu should include a dropdown component")
	}
	if len(fs.removedWords) != 0 {
		t.Error("opening the menu must not delete anything yet")
	}
}

func TestRemoveBannedWordCommandEmpty(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.removeBannedWord(fake, modInteraction("remove-banned-word"))

	if len(fake.followups) != 1 {
		t.Fatalf("expected 1 follow-up, got %d", len(fake.followups))
	}
	if len(fake.followups[0].Components) != 0 {
		t.Error("an empty list must not show a dropdown")
	}
	if !strings.Contains(fake.followups[0].Embeds[0].Description, "No ") {
		t.Error("empty list should show the 'none configured' message")
	}
}

func TestRemoveBannedWordSelectionDeletes(t *testing.T) {
	fs := &fakeStore{bannedWords: []database.BannedWord{{ID: 7, WordPattern: "spam"}}}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.handleRemoveSelection(fake, componentInteraction("rmword_select_0", "7"), d.bannedWordRemoveSelector(i18n.EN), "test")

	if len(fs.removedWords) != 1 || fs.removedWords[0] != 7 {
		t.Errorf("removedWords = %v, want [7]", fs.removedWords)
	}
	if fake.responds != 1 {
		t.Errorf("expected the menu to be refreshed once, got %d responses", fake.responds)
	}
}

func TestRemoveBannedWordSelectionIgnoresOtherComponents(t *testing.T) {
	fs := &fakeStore{bannedWords: []database.BannedWord{{ID: 7, WordPattern: "spam"}}}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	// A component belonging to a different selector must be left alone.
	d.handleRemoveSelection(fake, componentInteraction("rmsite_select_0", "7"), d.bannedWordRemoveSelector(i18n.EN), "test")

	if len(fs.removedWords) != 0 {
		t.Errorf("unrelated component must not delete words, got %v", fs.removedWords)
	}
	if fake.responds != 0 {
		t.Errorf("unrelated component must not be answered, got %d", fake.responds)
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
	if !strings.Contains(fake.followups[0].Embeds[0].Description, "No ") {
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

func TestRemoveBannedWebsiteCommandShowsMenu(t *testing.T) {
	fs := &fakeStore{bannedWebsites: []database.BannedWebsite{{ID: 3, WebsiteURL: "evil.com"}}}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.removeBannedWebsite(fake, modInteraction("remove-banned-website"))

	if len(fake.followups) != 1 {
		t.Fatalf("expected 1 follow-up, got %d", len(fake.followups))
	}
	if len(fake.followups[0].Components) == 0 {
		t.Error("the removal menu should include a dropdown component")
	}
}

func TestRemoveBannedWebsiteSelectionDeletes(t *testing.T) {
	fs := &fakeStore{bannedWebsites: []database.BannedWebsite{{ID: 3, WebsiteURL: "evil.com"}}}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.handleRemoveSelection(fake, componentInteraction("rmsite_select_0", "3"), d.bannedWebsiteRemoveSelector(i18n.EN), "test")

	if len(fs.removedWebsites) != 1 || fs.removedWebsites[0] != 3 {
		t.Errorf("removedWebsites = %v, want [3]", fs.removedWebsites)
	}
	if fake.responds != 1 {
		t.Errorf("expected the menu to be refreshed once, got %d responses", fake.responds)
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

func TestSelectModeratorAndExcludedChannelsCommands(t *testing.T) {
	channels := []*discordgo.Channel{{ID: "c1", Name: "general", Type: discordgo.ChannelTypeGuildText}}

	for _, tc := range []struct {
		name string
		call func(*Discord, discordClient, *discordgo.InteractionCreate)
	}{
		{"moderator", (*Discord).selectModeratorChannels},
		{"excluded", (*Discord).selectExcludedChannels},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newTestDiscordWith(&fakeStore{}, NewCache(100, 3, time.Hour))
			fake := &fakeSender{guildChannels: channels}

			tc.call(d, fake, modInteraction("set-channels"))

			if len(fake.followups) != 1 {
				t.Fatalf("expected the selection menu follow-up, got %d", len(fake.followups))
			}
		})
	}
}
