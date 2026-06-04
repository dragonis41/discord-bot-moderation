package database

import "testing"

func TestBannedWordsCRUD(t *testing.T) {
	d := newTestDB(t)

	// Empty to start.
	words, err := d.GetBannedWordsByGuildId("g1")
	if err != nil {
		t.Fatalf("GetBannedWordsByGuildId: %v", err)
	}
	if len(words) != 0 {
		t.Fatalf("expected no banned words, got %d", len(words))
	}

	if err := d.AddBannedWord("g1", "badword", false); err != nil {
		t.Fatalf("AddBannedWord: %v", err)
	}
	if err := d.AddBannedWord("g1", "ba.*d", true); err != nil {
		t.Fatalf("AddBannedWord (regex): %v", err)
	}
	// A word for another guild must not leak into g1.
	if err := d.AddBannedWord("g2", "other", false); err != nil {
		t.Fatalf("AddBannedWord (g2): %v", err)
	}

	words, err = d.GetBannedWordsByGuildId("g1")
	if err != nil {
		t.Fatalf("GetBannedWordsByGuildId: %v", err)
	}
	if len(words) != 2 {
		t.Fatalf("expected 2 banned words for g1, got %d", len(words))
	}

	// Verify the regex flag round-trips correctly.
	var sawRegex, sawLiteral bool
	for _, w := range words {
		if w.GuildID != "g1" {
			t.Errorf("got word for wrong guild: %q", w.GuildID)
		}
		switch w.WordPattern {
		case "ba.*d":
			sawRegex = w.IsRegex
		case "badword":
			sawLiteral = !w.IsRegex
		}
	}
	if !sawRegex {
		t.Error("regex word did not round-trip with IsRegex=true")
	}
	if !sawLiteral {
		t.Error("literal word did not round-trip with IsRegex=false")
	}

	// Remove one by ID.
	target := words[0]
	removed, err := d.RemoveBannedWord("g1", target.ID)
	if err != nil {
		t.Fatalf("RemoveBannedWord: %v", err)
	}
	if !removed {
		t.Error("RemoveBannedWord should report a deletion for an existing id")
	}
	words, _ = d.GetBannedWordsByGuildId("g1")
	if len(words) != 1 {
		t.Fatalf("expected 1 word after removal, got %d", len(words))
	}

	// Removing g2's id from g1 must delete nothing and report false rather than
	// a false success.
	g2Word, _ := d.GetBannedWordsByGuildId("g2")
	removed, err = d.RemoveBannedWord("g1", g2Word[0].ID)
	if err != nil {
		t.Fatalf("RemoveBannedWord (cross-guild): %v", err)
	}
	if removed {
		t.Error("RemoveBannedWord must not delete another guild's word")
	}

	// Remove all for guild.
	if err := d.RemoveBannedWordsByGuild("g1"); err != nil {
		t.Fatalf("RemoveBannedWordsByGuild: %v", err)
	}
	words, _ = d.GetBannedWordsByGuildId("g1")
	if len(words) != 0 {
		t.Fatalf("expected 0 words after RemoveBannedWordsByGuild, got %d", len(words))
	}

	// g2 must be untouched by g1 operations.
	g2Words, _ := d.GetBannedWordsByGuildId("g2")
	if len(g2Words) != 1 {
		t.Fatalf("expected g2 to still have 1 word, got %d", len(g2Words))
	}
}

func TestBannedWebsitesCRUD(t *testing.T) {
	d := newTestDB(t)

	if err := d.AddBannedWebsite("g1", "example.com"); err != nil {
		t.Fatalf("AddBannedWebsite: %v", err)
	}
	if err := d.AddBannedWebsite("g1", "evil.test"); err != nil {
		t.Fatalf("AddBannedWebsite: %v", err)
	}

	sites, err := d.GetBannedWebsitesByGuildId("g1")
	if err != nil {
		t.Fatalf("GetBannedWebsitesByGuildId: %v", err)
	}
	if len(sites) != 2 {
		t.Fatalf("expected 2 banned websites, got %d", len(sites))
	}

	removed, err := d.RemoveBannedWebsite("g1", sites[0].ID)
	if err != nil {
		t.Fatalf("RemoveBannedWebsite: %v", err)
	}
	if !removed {
		t.Error("RemoveBannedWebsite should report a deletion for an existing id")
	}
	sites, _ = d.GetBannedWebsitesByGuildId("g1")
	if len(sites) != 1 {
		t.Fatalf("expected 1 website after removal, got %d", len(sites))
	}

	// A non-existent id must report false rather than a false success.
	removed, err = d.RemoveBannedWebsite("g1", 999999)
	if err != nil {
		t.Fatalf("RemoveBannedWebsite (missing): %v", err)
	}
	if removed {
		t.Error("RemoveBannedWebsite must report false for a non-existent id")
	}

	if err := d.RemoveBannedWebsitesByGuild("g1"); err != nil {
		t.Fatalf("RemoveBannedWebsitesByGuild: %v", err)
	}
	sites, _ = d.GetBannedWebsitesByGuildId("g1")
	if len(sites) != 0 {
		t.Fatalf("expected 0 websites after RemoveBannedWebsitesByGuild, got %d", len(sites))
	}
}

func TestAutomoderationSettingsDefaultsAndPersistence(t *testing.T) {
	d := newTestDB(t)

	// With no row, all features default to enabled.
	settings, err := d.GetAutomoderationSettings("g1")
	if err != nil {
		t.Fatalf("GetAutomoderationSettings: %v", err)
	}
	if !settings.BannedWordsEnabled || !settings.BannedWebsitesEnabled || !settings.SpamDetectionEnabled {
		t.Fatalf("expected all features enabled by default, got %+v", settings)
	}
	if settings.GuildID != "g1" {
		t.Errorf("expected GuildID g1, got %q", settings.GuildID)
	}

	// Persist a mixed configuration.
	if err := d.SetAutomoderationSettings("g1", &AutomoderationSettings{
		GuildID:               "g1",
		BannedWordsEnabled:    false,
		BannedWebsitesEnabled: true,
		SpamDetectionEnabled:  false,
	}); err != nil {
		t.Fatalf("SetAutomoderationSettings: %v", err)
	}

	settings, _ = d.GetAutomoderationSettings("g1")
	if settings.BannedWordsEnabled {
		t.Error("BannedWordsEnabled should be false after update")
	}
	if !settings.BannedWebsitesEnabled {
		t.Error("BannedWebsitesEnabled should be true after update")
	}
	if settings.SpamDetectionEnabled {
		t.Error("SpamDetectionEnabled should be false after update")
	}

	// Upsert again to confirm ON CONFLICT updates rather than duplicating.
	if err := d.SetAutomoderationSettings("g1", &AutomoderationSettings{
		GuildID:               "g1",
		BannedWordsEnabled:    true,
		BannedWebsitesEnabled: false,
		SpamDetectionEnabled:  true,
	}); err != nil {
		t.Fatalf("SetAutomoderationSettings (upsert): %v", err)
	}
	settings, _ = d.GetAutomoderationSettings("g1")
	if !settings.BannedWordsEnabled || settings.BannedWebsitesEnabled || !settings.SpamDetectionEnabled {
		t.Fatalf("upsert did not apply expected values, got %+v", settings)
	}
}
