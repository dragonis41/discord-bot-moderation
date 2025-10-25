package api

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
)

func (d *Discord) messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from bots
	if m.Author.Bot {
		return
	}
	// Ignore messages from moderators
	if d.db.UserHasModerationRoleOnMessageCreate(s, m) {
		return
	}
	// TODO : Check if the message is sent in an excluded channel

	// Use the common moderation function
	d.moderateMessage(s, m.Message)
}

func (d *Discord) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {
	// Ignore messages from bots
	if m.Author != nil && m.Author.Bot {
		return
	}
	// Ignore messages from moderators
	if d.db.UserHasModerationRoleOnMessageUpdate(s, m) {
		return
	}
	// TODO : Check if the message is sent in an excluded channel

	// Use the common moderation function
	d.moderateMessage(s, m.Message)
}

// Common function that handles moderation logic
func (d *Discord) moderateMessage(s *discordgo.Session, m *discordgo.Message) {
	d.checkBannedWords(s, m)
	// TODO : Add more moderation checks
}

func (d *Discord) checkBannedWords(s *discordgo.Session, m *discordgo.Message) {
	// TODO: Get this list from database
	bannedWords := []string{"badword1", "badword2", "spam", `regex:spam[0-9]+`}

	messageLower := strings.ToLower(m.Content)

	for _, word := range bannedWords {
		var pattern string
		// Create regex pattern that matches the word only if it's surrounded by:
		// - start/end of string
		// - whitespace
		// - punctuation
		if strings.HasPrefix(word, "regex:") {
			// If the word starts with "regex:", treat the rest as a regex pattern
			regexPattern := strings.TrimPrefix(word, "regex:")
			pattern = fmt.Sprintf(`(?i)(^|[\s\p{P}])(%s)($|[\s\p{P}])`, regexPattern)
		} else {
			// Otherwise, escape the word to treat it as a literal string
			pattern = fmt.Sprintf(`(?i)(^|[\s\p{P}])%s($|[\s\p{P}])`, regexp.QuoteMeta(word))
		}
		matched, err := regexp.MatchString(pattern, messageLower)

		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "checkBannedWords()",
				Message:  fmt.Sprintf("Regex error for word '%s': %s", word, err),
			})
			continue
		}

		if matched {
			// Delete the message
			err := s.ChannelMessageDelete(m.ChannelID, m.ID)
			if err != nil {
				d.log.LogError(logger.LogModel{
					Database: d.db,
					GuildID:  m.GuildID,
					Function: "checkBannedWords()",
					Message:  fmt.Sprintf("Failed to delete message: %s", err),
				})
				return
			}

			// Send a warning to the user
			channel, err := s.UserChannelCreate(m.Author.ID)
			if err != nil {
				d.log.LogError(logger.LogModel{
					Database: d.db,
					GuildID:  m.GuildID,
					Function: "checkBannedWords()",
					Message:  fmt.Sprintf("Failed to create DM channel: %s", err),
				})
				return
			}

			guildName := "<unknown>"
			guild, err := d.client.Guild(m.GuildID)
			if err != nil {
				d.log.LogError(logger.LogModel{
					Database: d.db,
					GuildID:  m.GuildID,
					Function: "checkBannedWords()",
					Message:  fmt.Sprintf("Failed to fetch guild info: %s", err),
				})
			}
			if guild != nil {
				guildName = guild.Name
			}
			_, err = s.ChannelMessageSend(channel.ID, fmt.Sprintf("Votre message sur le server [%s] a été supprimé car il contient le mot `%s`\n\nVoici une copie de votre message :\n```\n%s\n```", guildName, word, m.Content))
			if err != nil {
				d.log.LogError(logger.LogModel{
					Database: d.db,
					GuildID:  m.GuildID,
					Function: "checkBannedWords()",
					Message:  fmt.Sprintf("Failed to send DM: %s", err),
				})
			}

			d.log.LogInfo(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "checkBannedWords()",
				Message:  fmt.Sprintf("Deleted message from %s containing forbidden word: %s", m.Author.Username, word),
			})

			return
		}
	}
}
