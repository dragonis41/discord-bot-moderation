package api

import (
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

// TestMain silences the standard logger so the package's tests don't spam the
// console with the bot's INFO/SUCCESS lines while exercising the effect layer.
func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	code := m.Run()
	log.SetOutput(os.Stderr)
	os.Exit(code)
}

// fakeStore satisfies the store interface by embedding it (a nil value): only the
// methods a test actually needs are implemented, and any unimplemented method
// would panic if unexpectedly called. The exported fields let each test configure
// the data the code under test reads, and the recorded slices let it assert on
// the writes.
type fakeStore struct {
	store

	// Configurable reads.
	settings       *database.AutomoderationSettings
	bannedWords    []database.BannedWord
	bannedWebsites []database.BannedWebsite
	isModerator    bool
	denyPermission bool
	excluded       map[string]bool
	excludedList   []string
	logChannels    []string
	modChannels    []string
	modRoles       []string
	sysLogCount    int
	sysLogMax      int
	sysLogEntries  []database.SystemLogEntry
	sysLogErrors   []database.SystemLogEntry
	modLogCount    int
	modLogMax      int
	modLogList     []database.ModerationLogEntry
	guildLang      string // language code returned by GetGuildLanguage ("" → default)

	// Recorded writes.
	savedLangs      []string
	modLogEntries   []modLogEntry
	addedWords      []addedWord
	removedWords    []int
	addedWebsites   []string
	removedWebsites []int
	savedSettings   []*database.AutomoderationSettings
	systemLogs      int
}

type modLogEntry struct {
	guildID  string
	action   model.ModerationLogAction
	userID   string
	username string
	trigger  string
	reason   string
}

type addedWord struct {
	pattern string
	isRegex bool
}

func (f *fakeStore) AddSystemLogEntry(string, model.SystemLogType, string, string) error {
	f.systemLogs++
	return nil
}

func (f *fakeStore) CheckModerationPermissionOnInteraction(database.InteractionResponder, *discordgo.InteractionCreate) bool {
	return !f.denyPermission
}

func (f *fakeStore) GetAutomoderationSettings(guildID string) (*database.AutomoderationSettings, error) {
	if f.settings != nil {
		return f.settings, nil
	}
	// Mirror the real default: everything enabled when no row exists.
	return &database.AutomoderationSettings{
		GuildID:               guildID,
		BannedWordsEnabled:    true,
		BannedWebsitesEnabled: true,
		SpamDetectionEnabled:  true,
	}, nil
}

func (f *fakeStore) SetAutomoderationSettings(_ string, s *database.AutomoderationSettings) error {
	f.savedSettings = append(f.savedSettings, s)
	return nil
}

func (f *fakeStore) GetBannedWordsByGuildId(string) ([]database.BannedWord, error) {
	return f.bannedWords, nil
}

func (f *fakeStore) GetBannedWebsitesByGuildId(string) ([]database.BannedWebsite, error) {
	return f.bannedWebsites, nil
}

func (f *fakeStore) AddBannedWord(_, wordPattern string, isRegex bool) error {
	f.addedWords = append(f.addedWords, addedWord{wordPattern, isRegex})
	return nil
}

func (f *fakeStore) RemoveBannedWord(_ string, id int) (bool, error) {
	f.removedWords = append(f.removedWords, id)
	return true, nil
}

func (f *fakeStore) AddBannedWebsite(_, websiteURL string) error {
	f.addedWebsites = append(f.addedWebsites, websiteURL)
	return nil
}

func (f *fakeStore) RemoveBannedWebsite(_ string, id int) (bool, error) {
	f.removedWebsites = append(f.removedWebsites, id)
	return true, nil
}

func (f *fakeStore) UserHasModerationRole(string, *discordgo.Member) bool {
	return f.isModerator
}

func (f *fakeStore) IsExcludedChannel(_ string, channelID string) bool {
	return f.excluded[channelID]
}

func (f *fakeStore) GetLogChannelsByGuildId(string) ([]string, error) {
	return f.logChannels, nil
}

func (f *fakeStore) GetModerationChannelsByGuildId(string) ([]string, error) {
	return f.modChannels, nil
}

func (f *fakeStore) GetModerationRolesByGuildId(string) ([]string, error) {
	return f.modRoles, nil
}

func (f *fakeStore) GetExcludedChannelsByGuildId(string) ([]string, error) {
	return f.excludedList, nil
}

func (f *fakeStore) AddModerationLogEntry(guildID string, action model.ModerationLogAction, userID, username, trigger, reason string) error {
	f.modLogEntries = append(f.modLogEntries, modLogEntry{guildID, action, userID, username, trigger, reason})
	return nil
}

func (f *fakeStore) GetGuildLanguage(string) (string, error) {
	if f.guildLang == "" {
		return "en", nil
	}
	return f.guildLang, nil
}

func (f *fakeStore) SetGuildLanguage(_, language string) error {
	f.savedLangs = append(f.savedLangs, language)
	return nil
}

func (f *fakeStore) GetSystemLogEntriesCount(string) (int, error)     { return f.sysLogCount, nil }
func (f *fakeStore) GetMaxSystemLogEntries(string) (int, error)       { return f.sysLogMax, nil }
func (f *fakeStore) GetModerationLogEntriesCount(string) (int, error) { return f.modLogCount, nil }
func (f *fakeStore) GetMaxModerationLogEntries(string) (int, error)   { return f.modLogMax, nil }

func (f *fakeStore) GetSystemLogEntriesByGuild(string, int) ([]database.SystemLogEntry, error) {
	return f.sysLogEntries, nil
}

func (f *fakeStore) GetSystemLogEntriesErrorsByGuildAndSystem(string, int) ([]database.SystemLogEntry, error) {
	return f.sysLogErrors, nil
}

func (f *fakeStore) GetModerationLogEntriesByGuild(string, int) ([]database.ModerationLogEntry, error) {
	return f.modLogList, nil
}

// fakeSender satisfies discordClient (and therefore discordSender) and records
// every side effect so tests can assert on them instead of talking to Discord.
type fakeSender struct {
	// Message-effect layer (discordSender).
	bans    []banCall
	kicks   []kickCall
	deletes []deleteCall
	dms     []dmCall
	embeds  []embedCall

	// Interaction layer (discordClient).
	responds      int
	followups     []*discordgo.WebhookParams
	followupEdits int
	responseEdits []*discordgo.WebhookEdit
	messageEdits  []*discordgo.MessageEdit

	// Configurable lookups.
	guildChannels []*discordgo.Channel
	guildRoles    []*discordgo.Role
}

type banCall struct {
	guildID, userID, reason string
	days                    int
}
type kickCall struct{ guildID, userID, reason string }
type deleteCall struct{ channelID, messageID string }
type dmCall struct{ recipientID, content string }
type embedCall struct {
	channelID string
	embed     *discordgo.MessageEmbed
}

// --- discordSender ---

func (f *fakeSender) Guild(guildID string, _ ...discordgo.RequestOption) (*discordgo.Guild, error) {
	return &discordgo.Guild{ID: guildID, Name: "TestGuild"}, nil
}

func (f *fakeSender) UserChannelCreate(recipientID string, _ ...discordgo.RequestOption) (*discordgo.Channel, error) {
	// Encode the recipient into the channel ID so ChannelMessageSend can recover it.
	return &discordgo.Channel{ID: "dm:" + recipientID}, nil
}

func (f *fakeSender) ChannelMessageSend(channelID, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	recipient := channelID
	if len(channelID) > 3 && channelID[:3] == "dm:" {
		recipient = channelID[3:]
	}
	f.dms = append(f.dms, dmCall{recipientID: recipient, content: content})
	return &discordgo.Message{}, nil
}

func (f *fakeSender) ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.embeds = append(f.embeds, embedCall{channelID: channelID, embed: embed})
	return &discordgo.Message{}, nil
}

func (f *fakeSender) ChannelMessageDelete(channelID, messageID string, _ ...discordgo.RequestOption) error {
	f.deletes = append(f.deletes, deleteCall{channelID: channelID, messageID: messageID})
	return nil
}

func (f *fakeSender) GuildMemberDeleteWithReason(guildID, userID, reason string, _ ...discordgo.RequestOption) error {
	f.kicks = append(f.kicks, kickCall{guildID: guildID, userID: userID, reason: reason})
	return nil
}

func (f *fakeSender) GuildBanCreateWithReason(guildID, userID, reason string, days int, _ ...discordgo.RequestOption) error {
	f.bans = append(f.bans, banCall{guildID: guildID, userID: userID, reason: reason, days: days})
	return nil
}

// --- discordClient ---

func (f *fakeSender) InteractionRespond(*discordgo.Interaction, *discordgo.InteractionResponse, ...discordgo.RequestOption) error {
	f.responds++
	return nil
}

func (f *fakeSender) InteractionResponseEdit(_ *discordgo.Interaction, newresp *discordgo.WebhookEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.responseEdits = append(f.responseEdits, newresp)
	return &discordgo.Message{}, nil
}

func (f *fakeSender) FollowupMessageCreate(_ *discordgo.Interaction, _ bool, data *discordgo.WebhookParams, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.followups = append(f.followups, data)
	return &discordgo.Message{ID: "followup"}, nil
}

func (f *fakeSender) FollowupMessageEdit(*discordgo.Interaction, string, *discordgo.WebhookEdit, ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.followupEdits++
	return &discordgo.Message{}, nil
}

func (f *fakeSender) GuildChannels(string, ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
	return f.guildChannels, nil
}

func (f *fakeSender) GuildRoles(string, ...discordgo.RequestOption) ([]*discordgo.Role, error) {
	return f.guildRoles, nil
}

func (f *fakeSender) User(userID string, _ ...discordgo.RequestOption) (*discordgo.User, error) {
	return &discordgo.User{ID: userID, Username: "user-" + userID}, nil
}

func (f *fakeSender) ChannelMessageEditComplex(m *discordgo.MessageEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.messageEdits = append(f.messageEdits, m)
	return &discordgo.Message{}, nil
}

// --- helpers ---

func newTestDiscord() *Discord {
	return newTestDiscordWith(&fakeStore{}, NewCache(1000, 3, time.Hour))
}

func newTestDiscordWith(db store, cache *Cache) *Discord {
	return &Discord{
		log:   logger.NewLogger(),
		db:    db,
		cache: cache,
	}
}

func testMessage() *discordgo.Message {
	return &discordgo.Message{
		ID:        "msg1",
		GuildID:   "g1",
		ChannelID: "c1",
		Content:   "offending content",
		Author:    &discordgo.User{ID: "u1", Username: "baduser"},
	}
}

// modInteraction builds an application-command interaction from a moderator,
// carrying the given command name and options.
func modInteraction(name string, opts ...*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			GuildID: "g1",
			Type:    discordgo.InteractionApplicationCommand,
			Member:  &discordgo.Member{User: &discordgo.User{ID: "mod1", Username: "moderator"}},
			Data: discordgo.ApplicationCommandInteractionData{
				Name:    name,
				Options: opts,
			},
		},
	}
}

func strOpt(name, value string) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{
		Name:  name,
		Type:  discordgo.ApplicationCommandOptionString,
		Value: value,
	}
}

func boolOpt(name string, value bool) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{
		Name:  name,
		Type:  discordgo.ApplicationCommandOptionBoolean,
		Value: value,
	}
}
