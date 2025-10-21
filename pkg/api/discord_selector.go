package api

import (
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

const channelsPerPage = 25
const rolesPerPage = 25

func (d *Discord) sendErrorMessage(s *discordgo.Session, i *discordgo.Interaction, title, description string) {
	_, _ = s.FollowupMessageCreate(i, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{{
			Title:       title,
			Description: description,
			Color:       model.Red.Int(),
			Timestamp:   time.Now().Format(time.RFC3339),
		}},
	})
}

func (d *Discord) getTextChannels(s *discordgo.Session, guildID string) ([]*discordgo.Channel, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return nil, err
	}

	var textChannels []*discordgo.Channel
	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildText {
			textChannels = append(textChannels, channel)
		}
	}

	// Sort channels by position
	sort.Slice(textChannels, func(i, j int) bool {
		return textChannels[i].Position < textChannels[j].Position
	})

	return textChannels, nil
}

func (d *Discord) getRoles(s *discordgo.Session, guildID string) ([]*discordgo.Role, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}

	// Sort roles by position
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Position < roles[j].Position
	})

	return roles, nil
}
