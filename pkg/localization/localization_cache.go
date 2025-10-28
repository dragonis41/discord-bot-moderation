package localization

import (
	"fmt"

	"github.com/dragonis41/discord-bot-moderation/internal/database"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// CacheInterface defines the interface for cache operations
// This avoids circular imports by not importing the actual Cache from api package
type CacheInterface interface {
	GetGuildLanguageCache(guildID string) (string, bool)
	SetGuildLanguageCache(guildID, language string)
	InvalidateGuildLanguageCache(guildID string)
}

// LocalizerWithCache manages localization with guild-specific language caching
type LocalizerWithCache struct {
	bundle *i18n.Bundle
	db     *database.Database
	cache  CacheInterface
}

// NewLocalizerWithCache creates a new localization manager with caching support
func NewLocalizerWithCache(bundle *i18n.Bundle, db *database.Database, cache CacheInterface) *LocalizerWithCache {
	return &LocalizerWithCache{
		bundle: bundle,
		db:     db,
		cache:  cache,
	}
}

// GetGuildLanguage retrieves the language for a guild, checking cache first
func (l *LocalizerWithCache) GetGuildLanguage(guildID string) (string, error) {
	// Check cache first
	if lang, exists := l.cache.GetGuildLanguageCache(guildID); exists {
		return lang, nil
	}

	// Fetch from database
	lang, err := l.db.GetGuildLanguage(guildID)
	if err != nil {
		return "", fmt.Errorf("failed to get guild language: %w", err)
	}

	// Store in cache
	l.cache.SetGuildLanguageCache(guildID, lang)

	return lang, nil
}

// SetGuildLanguage sets the language preference for a guild and updates cache
func (l *LocalizerWithCache) SetGuildLanguage(guildID, language string) error {
	// Update database
	if err := l.db.SetGuildLanguage(guildID, language); err != nil {
		return fmt.Errorf("failed to set guild language: %w", err)
	}

	// Update cache
	l.cache.SetGuildLanguageCache(guildID, language)

	return nil
}

// GetMessage retrieves a localized message for a guild
// This is used for user-facing messages sent to Discord
func (l *LocalizerWithCache) GetMessage(guildID, messageID string, templateData map[string]interface{}) (string, error) {
	// Get guild language
	lang, err := l.GetGuildLanguage(guildID)
	if err != nil {
		// Fallback to English on error
		lang = "en_US"
	}

	// Parse language tag
	tag, err := language.Parse(lang)
	if err != nil {
		tag = language.English
	}

	// Create localizer for this language
	localizer := i18n.NewLocalizer(l.bundle, tag.String())

	// Create message config
	messageConfig := &i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	}

	// Retrieve and localize message
	message, err := localizer.Localize(messageConfig)
	if err != nil {
		return "", fmt.Errorf("message not found: %s", messageID)
	}

	return message, nil
}

// GetMessagePlural retrieves a pluralized localized message for a guild
// This is used for messages that have different forms based on count
func (l *LocalizerWithCache) GetMessagePlural(guildID, messageID string, count int, templateData map[string]interface{}) (string, error) {
	// Get guild language
	lang, err := l.GetGuildLanguage(guildID)
	if err != nil {
		// Fallback to English on error
		lang = "en_US"
	}

	// Parse language tag
	tag, err := language.Parse(lang)
	if err != nil {
		tag = language.English
	}

	// Create localizer for this language
	localizer := i18n.NewLocalizer(l.bundle, tag.String())

	// Create message config for plural handling
	messageConfig := &i18n.LocalizeConfig{
		MessageID:    messageID,
		PluralCount:  count,
		TemplateData: templateData,
	}

	// Retrieve and localize message
	message, err := localizer.Localize(messageConfig)
	if err != nil {
		return "", fmt.Errorf("message not found: %s", messageID)
	}

	return message, nil
}

// GetSystemMessage retrieves a system/log message (always in default language, not localized)
// This is used for bot logs and system messages
func (l *LocalizerWithCache) GetSystemMessage(messageID string, templateData map[string]interface{}) (string, error) {
	// Always use English for system messages
	localizer := i18n.NewLocalizer(l.bundle, language.English.String())

	// Create message config
	messageConfig := &i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	}

	// Retrieve and localize message
	message, err := localizer.Localize(messageConfig)
	if err != nil {
		return "", fmt.Errorf("message not found: %s", messageID)
	}

	return message, nil
}
