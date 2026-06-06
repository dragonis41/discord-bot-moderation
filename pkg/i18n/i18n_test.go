package i18n

import "testing"

func TestTReturnsTranslation(t *testing.T) {
	if got := T(FR, "footer.hint"); got != fr["footer.hint"] {
		t.Errorf("T(FR, footer.hint) = %q, want the French string", got)
	}
	if got := T(ES, "footer.hint"); got != es["footer.hint"] {
		t.Errorf("T(ES, footer.hint) = %q, want the Spanish string", got)
	}
}

func TestTFormatsArgs(t *testing.T) {
	got := T(EN, "report.received", "alice")
	if got != "User alice has been reported to the moderation team." {
		t.Errorf("T with arg = %q", got)
	}
}

func TestTFallsBackToDefault(t *testing.T) {
	// A key present only in English must resolve for any language via fallback.
	const onlyEN = "test.only.english"
	en[onlyEN] = "english only"
	defer delete(en, onlyEN)

	if got := T(FR, onlyEN); got != "english only" {
		t.Errorf("expected fallback to English, got %q", got)
	}
}

func TestTUnknownKeyReturnsKey(t *testing.T) {
	if got := T(EN, "does.not.exist"); got != "does.not.exist" {
		t.Errorf("unknown key = %q, want the key itself", got)
	}
}

func TestParse(t *testing.T) {
	cases := map[string]Lang{"en": EN, "fr": FR, "es": ES, "": Default, "de": Default, "zz": Default}
	for in, want := range cases {
		if got := Parse(in); got != want {
			t.Errorf("Parse(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCatalogsCoverEnglishKeys guards against a translator forgetting a key:
// every key defined in English must also exist in French and Spanish.
func TestCatalogsCoverEnglishKeys(t *testing.T) {
	for _, lang := range []Lang{FR, ES} {
		for key := range en {
			if _, ok := catalog[lang][key]; !ok {
				t.Errorf("language %q is missing key %q", lang, key)
			}
		}
	}
}
