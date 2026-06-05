package api

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func chainConfig() SelectionConfig {
	return SelectionConfig{Prefix: "p", ItemsPerPage: 25, Title: "T", Description: "D"}
}

func chainFetcher(discordClient, string) ([]SelectionItem, error) {
	return []SelectionItem{
		AutomodFeatureItem{ID: "c1", Name: "C1"},
		AutomodFeatureItem{ID: "c2", Name: "C2"},
	}, nil
}

func componentInteraction(customID string, values ...string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			GuildID: "g1",
			Type:    discordgo.InteractionMessageComponent,
			Member:  &discordgo.Member{User: &discordgo.User{ID: "mod1", Username: "mod"}},
			Data:    discordgo.MessageComponentInteractionData{CustomID: customID, Values: values},
		},
	}
}

func TestHandleSelectionIgnoresOtherPrefix(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}
	dbOps := &fakeDBOps{}

	d.handleSelection(fake, componentInteraction("other_select_done"), chainConfig(), dbOps, chainFetcher, nil)

	if fake.responds != 0 {
		t.Error("a custom ID with a different prefix must be ignored before responding")
	}
}

func TestHandleSelectionIgnoresNonComponent(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}
	dbOps := &fakeDBOps{}

	// An application-command interaction is not a component interaction.
	d.handleSelection(fake, modInteraction("p"), chainConfig(), dbOps, chainFetcher, nil)

	if fake.responds != 0 {
		t.Error("non-component interactions must be ignored")
	}
}

func TestHandleSelectionUpdate(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}
	dbOps := &fakeDBOps{selected: nil}

	d.handleSelection(fake, componentInteraction("p_select_menu_0", "c1"), chainConfig(), dbOps, chainFetcher, nil)

	if fake.responds != 1 {
		t.Errorf("expected the interaction to be acknowledged once, got %d", fake.responds)
	}
	if dbOps.cleared != 1 {
		t.Errorf("selection update should clear before re-adding, cleared=%d", dbOps.cleared)
	}
	if len(dbOps.added) != 1 || dbOps.added[0] != "c1" {
		t.Errorf("expected c1 to be persisted, added=%v", dbOps.added)
	}
	if len(fake.responseEdits) != 1 {
		t.Errorf("expected the message to be re-rendered once, got %d edits", len(fake.responseEdits))
	}
}

func TestHandleSelectionPageNavigation(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}
	dbOps := &fakeDBOps{}

	d.handleSelection(fake, componentInteraction("p_page_next_0"), chainConfig(), dbOps, chainFetcher, nil)

	if fake.responds != 1 {
		t.Errorf("expected one acknowledgement, got %d", fake.responds)
	}
	if len(fake.responseEdits) != 1 {
		t.Errorf("page navigation should re-render the menu, got %d edits", len(fake.responseEdits))
	}
}

func TestHandleSelectionDone(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}
	dbOps := &fakeDBOps{selected: []string{"c1"}}
	formatter := func([]SelectionItem) string { return "all set" }

	d.handleSelection(fake, componentInteraction("p_select_done"), chainConfig(), dbOps, chainFetcher, formatter)

	if fake.responds != 1 {
		t.Errorf("expected one acknowledgement, got %d", fake.responds)
	}
	if len(fake.responseEdits) != 1 {
		t.Fatalf("done should edit the response once, got %d", len(fake.responseEdits))
	}
	// The components are cleared on completion.
	edit := fake.responseEdits[0]
	if edit.Components == nil || len(*edit.Components) != 0 {
		t.Error("completion should remove the menu components")
	}
}

func TestSendSelectPageEditsExistingMessage(t *testing.T) {
	d := newTestDiscord()
	fake := &fakeSender{}
	dbOps := &fakeDBOps{}
	items := []SelectionItem{AutomodFeatureItem{ID: "c1", Name: "C1"}}

	// An interaction that already has a message edits it instead of creating one.
	interaction := &discordgo.Interaction{GuildID: "g1", Message: &discordgo.Message{ID: "m1"}}
	d.sendSelectPage(fake, interaction, items, 0, chainConfig(), dbOps)

	if fake.followupEdits != 1 {
		t.Errorf("expected an existing message to be edited, followupEdits=%d", fake.followupEdits)
	}
	if len(fake.followups) != 0 {
		t.Errorf("expected no new follow-up to be created, got %d", len(fake.followups))
	}
}
