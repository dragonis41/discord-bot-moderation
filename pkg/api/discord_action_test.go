package api

import (
	"testing"

	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

// These tests exercise the message-effect layer (DM / warn / delete / kick / ban)
// against fake collaborators. The fakes live in fakes_test.go.

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

func TestGuildNameByID(t *testing.T) {
	fake := &fakeSender{}
	if got := guildNameByID(fake, "g1"); got != "TestGuild" {
		t.Errorf("guildNameByID = %q, want TestGuild", got)
	}
}
