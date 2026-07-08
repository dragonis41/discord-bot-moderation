package api

import (
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

// These tests exercise the message-effect layer (DM / warn / delete / kick / ban),
// the channel-broadcast helpers and the recent-message renderer in utils.go,
// against fake collaborators. The fakes live in setup_test.go.

func TestTakeAutomoderationActionBan(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}

	d.takeAutomoderationAction(fake, testMessage(), model.ActionBan, "spam")

	if len(fake.bans) != 1 {
		t.Fatalf("expected 1 ban, got %d", len(fake.bans))
	}
	b := fake.bans[0]
	if b.guildID != "g1" || b.userID != "u1" {
		t.Errorf("ban target = %s/%s, want g1/u1", b.guildID, b.userID)
	}
	if b.days != 1 {
		t.Errorf("ban days = %d, want 1 (delete 1 day of messages)", b.days)
	}
	// The user is warned by DM before the ban.
	if len(fake.dms) != 1 {
		t.Fatalf("expected 1 DM before ban, got %d", len(fake.dms))
	}
	if fake.dms[0].recipientID != "u1" {
		t.Errorf("DM recipient = %s, want u1", fake.dms[0].recipientID)
	}
	if len(fake.kicks) != 0 || len(fake.deletes) != 0 {
		t.Error("ban action should not kick or delete")
	}
}

func TestTakeAutomoderationActionKick(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}

	d.takeAutomoderationAction(fake, testMessage(), model.ActionKick, "reason")

	if len(fake.kicks) != 1 {
		t.Fatalf("expected 1 kick, got %d", len(fake.kicks))
	}
	if fake.kicks[0].userID != "u1" {
		t.Errorf("kick target = %s, want u1", fake.kicks[0].userID)
	}
	if len(fake.dms) != 1 {
		t.Errorf("expected the kicked user to be DM'd, got %d DMs", len(fake.dms))
	}
	if len(fake.bans) != 0 {
		t.Error("kick action should not ban")
	}
}

func TestTakeAutomoderationActionDeleteMessage(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}

	d.takeAutomoderationAction(fake, testMessage(), model.ActionDeleteMessage, "reason")

	if len(fake.deletes) != 1 {
		t.Fatalf("expected 1 delete, got %d", len(fake.deletes))
	}
	del := fake.deletes[0]
	if del.channelID != "c1" || del.messageID != "msg1" {
		t.Errorf("delete target = %s/%s, want c1/msg1", del.channelID, del.messageID)
	}
	// Deleting a message does not DM, ban or kick.
	if len(fake.dms) != 0 || len(fake.bans) != 0 || len(fake.kicks) != 0 {
		t.Error("delete action should not DM, ban or kick")
	}
}

func TestTakeAutomoderationActionWarn(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}

	d.takeAutomoderationAction(fake, testMessage(), model.ActionWarn, "be nice")

	if len(fake.dms) != 1 {
		t.Fatalf("expected 1 warning DM, got %d", len(fake.dms))
	}
	if fake.dms[0].content != "be nice" {
		t.Errorf("warning content = %q, want %q", fake.dms[0].content, "be nice")
	}
	if len(fake.bans) != 0 || len(fake.kicks) != 0 || len(fake.deletes) != 0 {
		t.Error("warn action should take no destructive action")
	}
}

func TestSendDM(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}

	d.sendDM(fake, "g1", "u42", "TestSendDM", "hello there")

	if len(fake.dms) != 1 {
		t.Fatalf("expected 1 DM, got %d", len(fake.dms))
	}
	if fake.dms[0].recipientID != "u42" || fake.dms[0].content != "hello there" {
		t.Errorf("DM = %+v, want recipient u42 / content 'hello there'", fake.dms[0])
	}
}

// --- Channel-broadcast helpers ----------------------------------------------

func TestSendLogChannelsEmbed(t *testing.T) {
	fs := &fakeStore{logChannels: []string{"log1", "log2"}}
	d := newTestDiscordWith(fs, NewCache(1000, 3, time.Hour))
	fake := &fakeSender{}

	d.sendLogChannelsEmbed(fake, "g1", &discordgo.MessageEmbed{Title: "hi"})

	if len(fake.embeds) != 2 {
		t.Fatalf("expected the embed sent to 2 log channels, got %d", len(fake.embeds))
	}
	if fake.embeds[0].channelID != "log1" || fake.embeds[1].channelID != "log2" {
		t.Errorf("embeds sent to wrong channels: %+v", fake.embeds)
	}
}

func TestSendModChannelsEmbed(t *testing.T) {
	fs := &fakeStore{modChannels: []string{"m1", "m2"}}
	d := newTestDiscordWith(fs, NewCache(1000, 3, time.Hour))
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

// --- Recent-message renderer -------------------------------------------------

func TestGetUserRecentMessagesStringEmpty(t *testing.T) {
	d := newTestDiscord()
	out := d.getUserRecentMessagesString("g1", &discordgo.User{ID: "u1"}, 10)
	if !strings.Contains(out, "No recent message") {
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
	if !strings.Contains(out, "<file>") {
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
