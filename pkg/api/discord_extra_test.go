package api

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
)

// --- Selector command siblings ----------------------------------------------

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

// --- AutomodSettingsDB DatabaseOperations -----------------------------------

func TestAutomodSettingsDBGetSelected(t *testing.T) {
	fs := &fakeStore{settings: &database.AutomoderationSettings{
		BannedWordsEnabled:   true,
		SpamDetectionEnabled: true,
		// BannedWebsitesEnabled stays false
	}}
	ops := &AutomodSettingsDB{db: fs}

	sel, err := ops.GetSelected("g1")
	if err != nil {
		t.Fatalf("GetSelected: %v", err)
	}
	got := map[string]bool{}
	for _, s := range sel {
		got[s] = true
	}
	if !got["banned_words"] || !got["spam_detection"] {
		t.Errorf("expected banned_words + spam_detection enabled, got %v", sel)
	}
	if got["banned_websites"] {
		t.Errorf("banned_websites should not be reported as enabled, got %v", sel)
	}
}

func TestAutomodSettingsDBAddEnablesFeature(t *testing.T) {
	fs := &fakeStore{settings: &database.AutomoderationSettings{}}
	ops := &AutomodSettingsDB{db: fs}

	if err := ops.Add("g1", "banned_words"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(fs.savedSettings) != 1 || !fs.savedSettings[0].BannedWordsEnabled {
		t.Errorf("Add should enable banned_words, savedSettings=%+v", fs.savedSettings)
	}
}

func TestAutomodSettingsDBRemoveByGuildDisablesAll(t *testing.T) {
	fs := &fakeStore{}
	ops := &AutomodSettingsDB{db: fs}

	if err := ops.RemoveByGuild("g1"); err != nil {
		t.Fatalf("RemoveByGuild: %v", err)
	}
	if len(fs.savedSettings) != 1 {
		t.Fatalf("expected one saved settings, got %d", len(fs.savedSettings))
	}
	s := fs.savedSettings[0]
	if s.BannedWordsEnabled || s.BannedWebsitesEnabled || s.SpamDetectionEnabled {
		t.Errorf("RemoveByGuild should disable everything, got %+v", s)
	}
}

// GetSelected for the channel/role wrappers is a thin pass-through; verify the
// delegation returns what the store holds.
func TestChannelAndRoleDBOpsGetSelected(t *testing.T) {
	fs := &fakeStore{
		logChannels: []string{"l1"},
		modChannels: []string{"m1"},
		excluded:    map[string]bool{}, // GetExcluded uses its own method below
		modRoles:    []string{"r1"},
	}

	if got, _ := (&LogChannelDB{db: fs}).GetSelected("g1"); len(got) != 1 || got[0] != "l1" {
		t.Errorf("LogChannelDB.GetSelected = %v, want [l1]", got)
	}
	if got, _ := (&ModChannelDB{db: fs}).GetSelected("g1"); len(got) != 1 || got[0] != "m1" {
		t.Errorf("ModChannelDB.GetSelected = %v, want [m1]", got)
	}
	if got, _ := (&ModRoleDB{db: fs}).GetSelected("g1"); len(got) != 1 || got[0] != "r1" {
		t.Errorf("ModRoleDB.GetSelected = %v, want [r1]", got)
	}
}

// --- Small effect helpers ----------------------------------------------------

func TestSendModChannelsEmbed(t *testing.T) {
	fs := &fakeStore{modChannels: []string{"m1", "m2"}}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.sendModChannelsEmbed(fake, "g1", &discordgo.MessageEmbed{Title: "x"})

	if len(fake.embeds) != 2 {
		t.Errorf("expected the embed sent to 2 mod channels, got %d", len(fake.embeds))
	}
}

func TestSendPrivateMessageOnInteraction(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{GuildID: "g1"}}

	d.sendPrivateMessageOnInteraction(fake, i, &discordgo.User{ID: "u9"}, "hi")

	if len(fake.dms) != 1 || fake.dms[0].recipientID != "u9" || fake.dms[0].content != "hi" {
		t.Errorf("DM = %+v, want recipient u9 / 'hi'", fake.dms)
	}
}

func TestSendErrorMessage(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}
	i := &discordgo.Interaction{GuildID: "g1"}

	d.sendErrorMessage(fake, i, "Title", "something failed")

	if len(fake.followups) != 1 {
		t.Fatalf("expected 1 error follow-up, got %d", len(fake.followups))
	}
	if len(fake.followups[0].Embeds) == 0 || fake.followups[0].Embeds[0].Title != "Title" {
		t.Error("error follow-up should carry the titled embed")
	}
}

// --- Logging wrappers --------------------------------------------------------

func TestLogHelpersPersistToStore(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))

	d.logError("g1", "fn", "an error %d", 1)
	d.logInfo("g1", "fn", "info")
	d.logWarning("g1", "fn", "warn")
	d.logSuccess("g1", "fn", "ok")

	if fs.systemLogs != 4 {
		t.Errorf("expected 4 persisted system log entries, got %d", fs.systemLogs)
	}
}
