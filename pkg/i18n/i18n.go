// Package i18n is a tiny, dependency-free translation layer for the bot.
//
// Each user-facing string lives under a stable key in a per-language catalog
// (see en.go / fr.go / es.go). Call sites resolve a guild's language once and
// then look strings up with T. English is the default and is guaranteed to hold
// every key, so it doubles as the fallback for any gap in another language.
package i18n

import "fmt"

// Lang is an ISO-639-1 language code the bot can render messages in.
type Lang string

const (
	EN Lang = "en"
	FR Lang = "fr"
	ES Lang = "es"
)

// Default is used when a guild has no language set and as the fallback when a
// key is missing from another language's catalog.
const Default = EN

// Supported lists the selectable languages, in the order shown in the /lang
// menu.
var Supported = []Lang{EN, FR, ES}

// T returns the translation of key for lang. It falls back to the Default
// language and finally to the raw key, so a missing translation degrades
// gracefully instead of rendering an empty string. When args are supplied the
// stored string is treated as an fmt format string.
func T(lang Lang, key string, args ...any) string {
	s, ok := lookup(lang, key)
	if !ok {
		if s, ok = lookup(Default, key); !ok {
			s = key
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

func lookup(lang Lang, key string) (string, bool) {
	m, ok := catalog[lang]
	if !ok {
		return "", false
	}
	s, ok := m[key]
	return s, ok
}

// Parse normalizes a stored language code into a supported Lang, returning
// Default for empty or unknown values.
func Parse(code string) Lang {
	switch Lang(code) {
	case EN:
		return EN
	case FR:
		return FR
	case ES:
		return ES
	default:
		return Default
	}
}

// Name returns the language's own native display name (for the /lang menu).
func Name(lang Lang) string {
	switch lang {
	case FR:
		return "Français"
	case ES:
		return "Español"
	default:
		return "English"
	}
}

// Flag returns a flag emoji representing the language (for the /lang menu).
func Flag(lang Lang) string {
	switch lang {
	case FR:
		return "🇫🇷"
	case ES:
		return "🇪🇸"
	default:
		return "🇬🇧"
	}
}
