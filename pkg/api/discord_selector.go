package api

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

const channelsPerPage = 25
const rolesPerPage = 25

func (d *Discord) sendErrorMessage(s *discordgo.Session, i *discordgo.Interaction, title, description string) {
	_, _ = s.FollowupMessageCreate(i, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{{
			Title:       title,
			Description: description,
			Color:       red,
			Timestamp:   time.Now().Format(time.RFC3339),
		}},
	})
}
