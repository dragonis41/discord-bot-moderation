package api

import (
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

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

func TestGetUserRecentMessagesStringEmpty(t *testing.T) {
	d := newTestDiscord()
	out := d.getUserRecentMessagesString("g1", &discordgo.User{ID: "u1"}, 10)
	if !strings.Contains(out, "Aucun message") {
		t.Errorf("empty history = %q, want the 'no messages' placeholder", out)
	}
}

func TestGetUserRecentMessagesStringFormatsContent(t *testing.T) {
	d := newTestDiscord()
	d.cache.AddMessage(msg("g1", "c1", "m1", "u1", "hello there"))

	out := d.getUserRecentMessagesString("g1", &discordgo.User{ID: "u1"}, 10)
	if !strings.Contains(out, "hello there") {
		t.Errorf("expected the message content in the output, got %q", out)
	}
	if !strings.Contains(out, "<#c1>") {
		t.Errorf("expected the channel reference in the output, got %q", out)
	}
}

func TestGetUserRecentMessagesStringAttachmentPlaceholder(t *testing.T) {
	d := newTestDiscord()
	// A message with no text but one attachment.
	d.cache.AddMessage(&discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID: "m1", GuildID: "g1", ChannelID: "c1", Content: "",
			Author:      &discordgo.User{ID: "u1"},
			Attachments: []*discordgo.MessageAttachment{{Filename: "pic.png"}},
		},
	})

	out := d.getUserRecentMessagesString("g1", &discordgo.User{ID: "u1"}, 10)
	if !strings.Contains(out, "<fichier>") {
		t.Errorf("expected attachment placeholder, got %q", out)
	}
}

func TestGetUserRecentMessagesStringTruncatesLongContent(t *testing.T) {
	d := newTestDiscord()
	long := strings.Repeat("a", 500)
	d.cache.AddMessage(msg("g1", "c1", "m1", "u1", long))

	out := d.getUserRecentMessagesString("g1", &discordgo.User{ID: "u1"}, 10)
	// Per-message cap is 300 chars; the line must be shorter than the raw content.
	if strings.Contains(out, long) {
		t.Error("overly long message should have been truncated")
	}
	if !strings.Contains(out, "...") {
		t.Error("truncated content should end with an ellipsis")
	}
}

func TestSendLogChannelsEmbed(t *testing.T) {
	fs := &fakeStore{logChannels: []string{"log1", "log2"}}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.sendLogChannelsEmbed(fake, "g1", &discordgo.MessageEmbed{Title: "hi"})

	if len(fake.embeds) != 2 {
		t.Fatalf("expected the embed sent to 2 log channels, got %d", len(fake.embeds))
	}
	if fake.embeds[0].channelID != "log1" || fake.embeds[1].channelID != "log2" {
		t.Errorf("embeds sent to wrong channels: %+v", fake.embeds)
	}
}
