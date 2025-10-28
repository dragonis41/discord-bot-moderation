package localization

import (
	"encoding/json"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type LocaleInterface interface {
	InitLocalization() *i18n.Bundle
}

type Locale struct{}

func NewLocale() *Locale {
	return &Locale{}
}

func (l *Locale) InitLocalization() *i18n.Bundle {
	// Initialize i18n bundle with English as the default language
	bundle := i18n.NewBundle(language.English)

	// Register JSON unmarshal function
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load translation files
	_, _ = bundle.LoadMessageFile("resources/en_US.json")
	_, _ = bundle.LoadMessageFile("resources/fr_FR.json")

	return bundle
}
