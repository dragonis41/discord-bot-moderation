package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

// Translation helpers -------------------------------------------------------

// lang resolves the language configured for a guild, falling back to the
// default language if the lookup fails.
func (d *Discord) lang(guildID string) i18n.Lang {
	code, err := d.db.GetGuildLanguage(guildID)
	if err != nil {
		d.logWarning(guildID, "lang()", "Failed to fetch guild language, using default: %s", err)
		return i18n.Default
	}
	return i18n.Parse(code)
}

// Logging helpers -----------------------------------------------------------
//
// These thin wrappers remove the repetitive logger.LogModel{...} boilerplate
// that previously surrounded every log call in this package. The database and
// guild ID are always attached so log entries are persisted per guild.

func (d *Discord) logError(guildID, function, format string, args ...any) {
	d.log.LogError(logger.LogModel{Database: d.db, GuildID: guildID, Function: function, Message: fmt.Sprintf(format, args...)})
}

func (d *Discord) logInfo(guildID, function, format string, args ...any) {
	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: guildID, Function: function, Message: fmt.Sprintf(format, args...)})
}

func (d *Discord) logSuccess(guildID, function, format string, args ...any) {
	d.log.LogSuccess(logger.LogModel{Database: d.db, GuildID: guildID, Function: function, Message: fmt.Sprintf(format, args...)})
}

func (d *Discord) logWarning(guildID, function, format string, args ...any) {
	d.log.LogWarning(logger.LogModel{Database: d.db, GuildID: guildID, Function: function, Message: fmt.Sprintf(format, args...)})
}

// Embed helpers -------------------------------------------------------------
//
// The vast majority of embeds sent by command handlers share the same shape:
// a title, a description, a color, the default footer and a UTC timestamp.

func embed(lang i18n.Lang, title, description string, color model.Color) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color.Int(),
		Footer:      hintFooter(lang),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

func errorEmbed(lang i18n.Lang, title, description string) *discordgo.MessageEmbed {
	return embed(lang, title, description, model.Red)
}

func successEmbed(lang i18n.Lang, title, description string) *discordgo.MessageEmbed {
	return embed(lang, title, description, model.Green)
}

func infoEmbed(lang i18n.Lang, title, description string) *discordgo.MessageEmbed {
	return embed(lang, title, description, model.Blue)
}

// hintFooter returns the localized "use /help" footer shown on most embeds.
func hintFooter(lang i18n.Lang) *discordgo.MessageEmbedFooter {
	return &discordgo.MessageEmbedFooter{Text: i18n.T(lang, "footer.hint")}
}

// selectionFooter returns the localized footer shown on live selection menus.
func selectionFooter(lang i18n.Lang) *discordgo.MessageEmbedFooter {
	return &discordgo.MessageEmbedFooter{Text: i18n.T(lang, "footer.selection_saved")}
}

// truncate shortens s to at most max characters, appending "..." when it had to
// cut anything off.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// sanitizeInlineCode makes user content safe to display inside a single-line
// Discord inline-code span (`...`). Backticks would otherwise terminate the span
// and break the surrounding formatting, and newlines can't live inside inline
// code, so both are neutralized: backticks become a look-alike grave accent and
// line breaks become spaces.
func sanitizeInlineCode(s string) string {
	replacer := strings.NewReplacer(
		"`", "ˋ", // U+02CB, visually similar to a backtick but inert
		"\n", " ",
		"\r", " ",
	)
	return replacer.Replace(s)
}

// joinLinesWithLimit concatenates lines in order until adding the next one would
// exceed maxLen, at which point it appends a truncation notice and stops. It is
// used to keep log dumps below Discord's per-field length limit.
func joinLinesWithLimit(lines []string, maxLen int) string {
	out := ""
	for _, line := range lines {
		if len(out)+len(line) > maxLen {
			out += "\n... (logs truncated)"
			break
		}
		out += line
	}
	return out
}

// Interaction helpers -------------------------------------------------------

// deferEphemeral acknowledges an interaction with a deferred, ephemeral
// response. It returns false (after logging) if the acknowledgement failed, in
// which case the caller must stop processing.
func (d *Discord) deferEphemeral(s discordClient, i *discordgo.InteractionCreate, function string) bool {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})
	if err != nil {
		d.logError(i.GuildID, function, "Error deferring response: %s", err)
		return false
	}
	return true
}

// logCommand records that a slash command was received from a user.
func (d *Discord) logCommand(i *discordgo.InteractionCreate, function string) {
	d.logInfo(i.GuildID, function, "Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username)
}

// beginModCommand runs the common preamble shared by every moderator-only
// command: defer the response, log the invocation and verify permissions. It
// returns false if the caller should stop (failed defer or missing permission).
func (d *Discord) beginModCommand(s discordClient, i *discordgo.InteractionCreate, function string) bool {
	if !d.deferEphemeral(s, i, function) {
		return false
	}
	d.logCommand(i, function)
	return d.db.CheckModerationPermissionOnInteraction(s, i)
}

// followup sends an ephemeral follow-up message containing the given embeds,
// logging any send error under function.
func (d *Discord) followup(s discordClient, i *discordgo.InteractionCreate, function string, embeds ...*discordgo.MessageEmbed) {
	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Embeds: embeds}); err != nil {
		d.logError(i.GuildID, function, "Error sending follow-up message: %s", err)
	}
}

// Guild helpers -------------------------------------------------------------

// guildDisplayName returns guild.Name, fetching the full guild from the API
// when the cached state has an empty name. It never returns an empty string.
func (d *Discord) guildDisplayName(guild *discordgo.Guild) string {
	if guild.Name != "" {
		return guild.Name
	}
	full, err := d.client.Guild(guild.ID)
	if err != nil || full == nil {
		return "<error fetching>"
	}
	return full.Name
}

// guildNameByID safely resolves a guild's name from its ID, returning a
// placeholder if the guild cannot be fetched (instead of panicking on a nil
// guild as the previous code did).
func guildNameByID(s discordSender, guildID string) string {
	guild, err := s.Guild(guildID)
	if err != nil || guild == nil {
		return "<unknown>"
	}
	return guild.Name
}
