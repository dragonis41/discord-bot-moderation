package api

import (
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// MessageCache stores recent messages for spam detection
type MessageCache struct {
	GuildID         string
	ChannelID       string
	MessageID       string
	Content         string
	AuthorID        string
	AuthorUsername  string
	Timestamp       time.Time
	AttachmentCount int
	HasEmbeds       bool
}

// UserViolation tracks banned word violations per user per guild
type UserViolation struct {
	UserID         string
	GuildID        string
	ViolationCount int
	LastViolation  time.Time
}

// Cache handles message caching and violation tracking
type Cache struct {
	// messageCache stores the last N messages per guild in chronological order
	// Key: guildID, Value: slice of MessageCache (oldest to newest)
	messageCache map[string][]MessageCache

	// messageCacheIndex provides O(1) lookup for message updates
	// Key: guildID, Value: map of messageID to its index in messageCache slice
	messageCacheIndex map[string]map[string]int

	// messageCacheMu protects concurrent access to messageCache and messageCacheIndex
	messageCacheMu sync.RWMutex

	// userViolations tracks banned word violations per user per guild
	// Key: "guildID:userID", Value: pointer to UserViolation
	userViolations map[string]*UserViolation

	// userViolationsMu protects concurrent access to userViolations
	userViolationsMu sync.RWMutex

	// maxCacheSize defines the maximum number of messages to store per guild
	maxCacheSize int

	// violationThreshold is the number of violations before notifying moderators
	violationThreshold int

	// violationWindow is the time period after which violation counts reset
	violationWindow time.Duration
}

// NewCache creates a new Cache instance
func NewCache(maxCacheSize, violationThreshold int, violationWindow time.Duration) *Cache {
	return &Cache{
		messageCache:       make(map[string][]MessageCache),
		messageCacheIndex:  make(map[string]map[string]int),
		userViolations:     make(map[string]*UserViolation),
		maxCacheSize:       maxCacheSize,
		violationThreshold: violationThreshold,
		violationWindow:    violationWindow,
	}
}

// GetMaxCacheSize returns the maximum cache size
func (c *Cache) GetMaxCacheSize() int {
	return c.maxCacheSize
}

// GetViolationThreshold returns the violation threshold
func (c *Cache) GetViolationThreshold() int {
	return c.violationThreshold
}

// GetViolationWindow returns the violation window duration
func (c *Cache) GetViolationWindow() time.Duration {
	return c.violationWindow
}

// AddMessage adds a message to the guild's cache (max messages defined by maxCacheSize)
func (c *Cache) AddMessage(m *discordgo.MessageCreate) {
	c.messageCacheMu.Lock()
	defer c.messageCacheMu.Unlock()

	cache := c.messageCache[m.GuildID]

	// Initialize index map for this guild if needed
	if c.messageCacheIndex[m.GuildID] == nil {
		c.messageCacheIndex[m.GuildID] = make(map[string]int)
	}

	// Add new message
	newIndex := len(cache)
	cache = append(cache, MessageCache{
		GuildID:         m.GuildID,
		ChannelID:       m.ChannelID,
		MessageID:       m.ID,
		Content:         m.Content,
		AuthorID:        m.Author.ID,
		AuthorUsername:  m.Author.Username,
		Timestamp:       time.Now(),
		AttachmentCount: len(m.Attachments),
		HasEmbeds:       len(m.Embeds) > 0,
	})

	// Update index
	c.messageCacheIndex[m.GuildID][m.ID] = newIndex

	// Keep only last N messages
	if len(cache) > c.maxCacheSize {
		// Remove oldest message from index
		oldestMsg := cache[0]
		delete(c.messageCacheIndex[m.GuildID], oldestMsg.MessageID)

		// Remove from slice
		cache = cache[1:]

		// Update all indices (shift by -1)
		for msgID := range c.messageCacheIndex[m.GuildID] {
			c.messageCacheIndex[m.GuildID][msgID]--
		}
	}

	c.messageCache[m.GuildID] = cache
}

// UpdateMessage updates a message in the guild's cache
func (c *Cache) UpdateMessage(m *discordgo.MessageUpdate) {
	c.messageCacheMu.Lock()
	defer c.messageCacheMu.Unlock()

	indexMap, exists := c.messageCacheIndex[m.GuildID]
	if !exists {
		return
	}

	index, found := indexMap[m.ID]
	if !found {
		return
	}

	cache := c.messageCache[m.GuildID]
	if index >= 0 && index < len(cache) {
		// Update content and timestamp
		cache[index].Content = m.Content
		cache[index].Timestamp = time.Now()
		// Update attachment info if available in the update event
		if m.Attachments != nil {
			cache[index].AttachmentCount = len(m.Attachments)
		}
		if m.Embeds != nil {
			cache[index].HasEmbeds = len(m.Embeds) > 0
		}
		c.messageCache[m.GuildID] = cache
	}
}

// IncrementViolation increments violation count and returns the new count
func (c *Cache) IncrementViolation(guildID, userID string) int {
	c.userViolationsMu.Lock()
	defer c.userViolationsMu.Unlock()

	key := guildID + ":" + userID
	violation, exists := c.userViolations[key]

	now := time.Now()

	if !exists {
		// First violation
		c.userViolations[key] = &UserViolation{
			UserID:         userID,
			GuildID:        guildID,
			ViolationCount: 1,
			LastViolation:  now,
		}
		return 1
	}

	// Check if violation window has expired
	if now.Sub(violation.LastViolation) > c.violationWindow {
		// Reset counter
		violation.ViolationCount = 1
		violation.LastViolation = now
		return 1
	}

	// Increment existing violation
	violation.ViolationCount++
	violation.LastViolation = now
	return violation.ViolationCount
}

// GetUserRecentMessages retrieves recent messages from a specific user
func (c *Cache) GetUserRecentMessages(guildID, userID string, limit int) []MessageCache {
	c.messageCacheMu.RLock()
	defer c.messageCacheMu.RUnlock()

	cache, exists := c.messageCache[guildID]
	if !exists {
		return nil
	}

	var userMessages []MessageCache
	// Iterate backwards to get most recent first
	for i := len(cache) - 1; i >= 0 && len(userMessages) < limit; i-- {
		if cache[i].AuthorID == userID {
			userMessages = append(userMessages, cache[i])
		}
	}

	return userMessages
}

func (c *Cache) GetGuildRecentMessages(guildID string, limit int) []MessageCache {
	c.messageCacheMu.RLock()
	defer c.messageCacheMu.RUnlock()

	cache, exists := c.messageCache[guildID]
	if !exists {
		return nil
	}

	var recentMessages []MessageCache
	// Iterate backwards to get most recent first
	for i := len(cache) - 1; i >= 0 && len(recentMessages) < limit; i-- {
		recentMessages = append(recentMessages, cache[i])
	}

	return recentMessages
}

// ResetViolations resets violations for a specific user in a guild
func (c *Cache) ResetViolations(guildID, userID string) {
	c.userViolationsMu.Lock()
	defer c.userViolationsMu.Unlock()

	key := guildID + ":" + userID
	delete(c.userViolations, key)
}

// GetViolationCount returns the current violation count for a user
func (c *Cache) GetViolationCount(guildID, userID string) int {
	c.userViolationsMu.RLock()
	defer c.userViolationsMu.RUnlock()

	key := guildID + ":" + userID
	violation, exists := c.userViolations[key]
	if !exists {
		return 0
	}

	// Check if violation window has expired
	if time.Now().Sub(violation.LastViolation) > c.violationWindow {
		return 0
	}

	return violation.ViolationCount
}
