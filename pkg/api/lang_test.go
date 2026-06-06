package api

import (
	"strings"
	"testing"
	"time"

	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
)

func TestSelectLanguageShowsMenu(t *testing.T) {
	fs := &fakeStore{guildLang: "fr"}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.selectLanguage(fake, modInteraction("lang"))

	if len(fake.followups) != 1 {
		t.Fatalf("expected 1 follow-up, got %d", len(fake.followups))
	}
	if len(fake.followups[0].Components) == 0 {
		t.Error("the /lang follow-up should carry the language select menu")
	}
	// Menu is rendered in the guild's current language (French).
	if title := fake.followups[0].Embeds[0].Title; title != i18n.T(i18n.FR, "lang.menu_title") {
		t.Errorf("menu title = %q, want the French title", title)
	}
}

func TestHandleLanguageSelectionPersistsAndConfirms(t *testing.T) {
	fs := &fakeStore{guildLang: "en"}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.applyLanguageSelection(fake, componentInteraction(langSelectCustomID, "es"))

	if len(fs.savedLangs) != 1 || fs.savedLangs[0] != "es" {
		t.Fatalf("savedLangs = %v, want [es]", fs.savedLangs)
	}
	if fake.responds != 1 {
		t.Errorf("expected the menu to be replaced once, got %d responses", fake.responds)
	}
}

func TestHandleLanguageSelectionIgnoresOtherComponents(t *testing.T) {
	fs := &fakeStore{}
	d := newTestDiscordWith(fs, NewCache(100, 3, time.Hour))
	fake := &fakeSender{}

	d.applyLanguageSelection(fake, componentInteraction("not_lang", "es"))

	if len(fs.savedLangs) != 0 {
		t.Errorf("unrelated component must not change the language, got %v", fs.savedLangs)
	}
	if fake.responds != 0 {
		t.Error("unrelated component must be ignored before responding")
	}
}

func TestLangUpdatedMessageUsesNewLanguage(t *testing.T) {
	// The confirmation must be written in the newly chosen language.
	msg := i18n.T(i18n.ES, "lang.updated", i18n.Name(i18n.ES))
	if !strings.Contains(msg, "Español") {
		t.Errorf("Spanish confirmation = %q, want it to name the language", msg)
	}
}
