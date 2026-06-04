package api

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// fakeDBOps is a configurable DatabaseOperations for selector tests.
type fakeDBOps struct {
	selected []string
	added    []string
	cleared  int
}

func (f *fakeDBOps) GetSelected(string) ([]string, error) { return f.selected, nil }
func (f *fakeDBOps) RemoveByGuild(string) error           { f.cleared++; return nil }
func (f *fakeDBOps) Add(_, itemID string) error           { f.added = append(f.added, itemID); return nil }

func TestBuildSelectMenuOptions(t *testing.T) {
	d := newTestDiscord()
	items := []SelectionItem{
		AutomodFeatureItem{ID: "a", Name: "Feature A", Description: "short"},
		AutomodFeatureItem{ID: "b", Name: "Feature B", Description: strings.Repeat("x", 150)},
	}

	opts := d.buildSelectMenuOptions(items, []string{"a"}, "🔧")
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}
	if !opts[0].Default {
		t.Error("option 'a' should be marked Default (it is in selectedIDs)")
	}
	if opts[1].Default {
		t.Error("option 'b' should not be Default")
	}
	// Descriptions over 100 chars are truncated to 100.
	if len(opts[1].Description) != 100 {
		t.Errorf("long description = %d chars, want 100", len(opts[1].Description))
	}
	if opts[0].Emoji == nil || opts[0].Emoji.Name != "🔧" {
		t.Error("emoji should be applied when emojiName is set")
	}
}

func TestBuildNavigationButtons(t *testing.T) {
	d := newTestDiscord()

	// Single page: only the "done" button.
	single := d.buildNavigationButtons(0, 1, "p")
	if len(single) != 1 {
		t.Fatalf("single page = %d buttons, want 1 (done only)", len(single))
	}

	// Multi page, first page: prev + indicator + next + done.
	first := d.buildNavigationButtons(0, 3, "p")
	if len(first) != 4 {
		t.Fatalf("multi page = %d buttons, want 4", len(first))
	}
	if !first[0].(discordgo.Button).Disabled {
		t.Error("prev button should be disabled on the first page")
	}
	if first[2].(discordgo.Button).Disabled {
		t.Error("next button should be enabled on the first page")
	}

	// Multi page, last page: next disabled.
	last := d.buildNavigationButtons(2, 3, "p")
	if !last[2].(discordgo.Button).Disabled {
		t.Error("next button should be disabled on the last page")
	}
}

func TestBuildSelectMessageClampsPage(t *testing.T) {
	d := newTestDiscord()
	items := []SelectionItem{
		AutomodFeatureItem{ID: "a", Name: "A"},
		AutomodFeatureItem{ID: "b", Name: "B"},
	}
	config := SelectionConfig{
		Prefix:       "p",
		ItemsPerPage: 25,
		Title:        "Title",
		Description:  "Desc",
	}
	dbOps := &fakeDBOps{selected: []string{"a"}}

	// Page 99 is out of range and must clamp without panicking.
	embed, components := d.buildSelectMessage("g1", items, 99, config, dbOps)
	if embed.Title != "Title" {
		t.Errorf("embed title = %q, want Title", embed.Title)
	}
	if !strings.Contains(embed.Description, "1") {
		t.Errorf("description should report 1 selected item: %q", embed.Description)
	}
	// Two rows: the select menu, plus the row that always carries the "done"
	// button (navigation arrows are omitted for a single page).
	if len(components) != 2 {
		t.Errorf("expected 2 component rows (menu + done), got %d", len(components))
	}
}

func TestSelectorConfigGetters(t *testing.T) {
	d := newTestDiscord()

	cases := []struct {
		name   string
		get    func() (SelectionConfig, DatabaseOperations)
		prefix string
	}{
		{"log", d.getLogChannelConfig, "log_channel"},
		{"mod", d.getModChannelConfig, "mod_channel"},
		{"excluded", d.getExcludedChannelConfig, "excluded_channel"},
		{"role", d.getModRoleConfig, "mod_role"},
		{"automod", d.getAutomodSettingsConfig, "automod"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config, dbOps := tc.get()
			if config.Prefix != tc.prefix {
				t.Errorf("prefix = %q, want %q", config.Prefix, tc.prefix)
			}
			if dbOps == nil {
				t.Error("dbOps should not be nil")
			}
		})
	}
}

func TestGetAutomoderationFeaturesAsItems(t *testing.T) {
	d := newTestDiscord()
	items, err := d.getAutomoderationFeaturesAsItems(nil, "g1")
	if err != nil {
		t.Fatalf("getAutomoderationFeaturesAsItems: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 features, got %d", len(items))
	}
	ids := map[string]bool{}
	for _, it := range items {
		ids[it.GetID()] = true
	}
	for _, want := range []string{"banned_words", "banned_websites", "spam_detection"} {
		if !ids[want] {
			t.Errorf("missing feature %q", want)
		}
	}
}

func TestChannelSelectionDoneMessage(t *testing.T) {
	formatter := channelSelectionDoneMessage("Header:")
	items := []SelectionItem{
		ChannelItem{&discordgo.Channel{ID: "c2", Position: 2}},
		ChannelItem{&discordgo.Channel{ID: "c1", Position: 1}},
	}
	out := formatter(items)

	if !strings.HasPrefix(out, "Header:") {
		t.Errorf("output should start with the header: %q", out)
	}
	// Sorted by position, so c1 (pos 1) comes before c2 (pos 2).
	if strings.Index(out, "c1") > strings.Index(out, "c2") {
		t.Errorf("channels should be sorted by position: %q", out)
	}
}

func TestSelectionItemAccessors(t *testing.T) {
	ch := ChannelItem{&discordgo.Channel{ID: "c1", Name: "general", Topic: "chat"}}
	if ch.GetID() != "c1" || ch.GetName() != "general" || ch.GetDescription() != "chat" {
		t.Errorf("ChannelItem accessors wrong: %s/%s/%s", ch.GetID(), ch.GetName(), ch.GetDescription())
	}

	role := RoleItem{&discordgo.Role{ID: "r1", Name: "Mods"}}
	if role.GetID() != "r1" || role.GetName() != "Mods" || role.GetDescription() != "" {
		t.Errorf("RoleItem accessors wrong: %s/%s/%q", role.GetID(), role.GetName(), role.GetDescription())
	}

	feat := AutomodFeatureItem{ID: "f", Name: "Feat", Description: "desc"}
	if feat.GetID() != "f" || feat.GetName() != "Feat" || feat.GetDescription() != "desc" {
		t.Error("AutomodFeatureItem accessors wrong")
	}
}
