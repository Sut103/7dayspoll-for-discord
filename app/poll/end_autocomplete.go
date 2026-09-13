package poll

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	// Discord's per-request maximum for ChannelMessages.
	scanLimit = 100
	// Discord's per-response maximums for autocomplete.
	choiceLimit     = 25
	nameLengthLimit = 100
	// Discord fires an autocomplete interaction on every keystroke with no debounce.
	cacheTTL = 10 * time.Second
)

type candidate struct {
	messageID string
	title     string
}

type cacheEntry struct {
	candidates []candidate
	fetchedAt  time.Time
}

type candidateCache struct {
	ttl time.Duration

	mu      sync.Mutex
	entries map[string]cacheEntry
	// Per channel, so one burst issues a single fetch.
	locks map[string]*sync.Mutex
}

func newCandidateCache(ttl time.Duration) *candidateCache {
	return &candidateCache{
		ttl:     ttl,
		entries: make(map[string]cacheEntry),
		locks:   make(map[string]*sync.Mutex),
	}
}

var cache = newCandidateCache(cacheTTL)

func (c *candidateCache) mutexFor(channelID string) *sync.Mutex {
	c.mu.Lock()
	defer c.mu.Unlock()
	lock, ok := c.locks[channelID]
	if !ok {
		lock = &sync.Mutex{}
		c.locks[channelID] = lock
	}
	return lock
}

func (c *candidateCache) candidates(session pollSession, channelID, botUserID string) ([]candidate, error) {
	lock := c.mutexFor(channelID)
	lock.Lock()
	defer lock.Unlock()

	c.mu.Lock()
	entry, ok := c.entries[channelID]
	c.mu.Unlock()
	if ok && time.Since(entry.fetchedAt) < c.ttl {
		return entry.candidates, nil
	}

	messages, err := session.ChannelMessages(channelID, scanLimit, "", "", "")
	if err != nil {
		return nil, err
	}

	var candidates []candidate
	for _, message := range messages {
		// A poll owned by another application would always fail to expire anyway.
		if message == nil || message.Poll == nil || message.Author == nil || message.Author.ID != botUserID {
			continue
		}
		if isFinalized(message.Poll) {
			continue
		}
		candidates = append(candidates, candidate{messageID: message.ID, title: message.Poll.Question.Text})
	}

	c.store(channelID, candidates)
	return candidates, nil
}

func (c *candidateCache) store(channelID string, candidates []candidate) {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[channelID] = cacheEntry{candidates: candidates, fetchedAt: now}
	// Idle channels must not pin memory for the bot's lifetime.
	for id, entry := range c.entries {
		if now.Sub(entry.fetchedAt) > 10*c.ttl {
			delete(c.entries, id)
			delete(c.locks, id)
		}
	}
}

func (c *candidateCache) size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func Autocomplete(session pollSession, interaction *discordgo.Interaction, botUserID string) error {
	return cache.autocomplete(session, interaction, botUserID)
}

func (c *candidateCache) autocomplete(session pollSession, interaction *discordgo.Interaction, botUserID string) error {
	options := interaction.ApplicationCommandData().Options
	if len(options) == 0 {
		return respondAutocomplete(session, interaction, nil)
	}
	query := strings.ToLower(strings.TrimSpace(options[0].StringValue()))

	candidates, err := c.candidates(session, interaction.ChannelID, botUserID)
	if err != nil {
		log.Println("Failed to list channel messages for poll-end autocomplete:", err)
		return respondAutocomplete(session, interaction, nil)
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, candidate := range candidates {
		if query != "" && !strings.Contains(strings.ToLower(candidate.title), query) {
			continue
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  truncateRunes(candidate.title, nameLengthLimit),
			Value: candidate.messageID,
		})
		if len(choices) == choiceLimit {
			break
		}
	}
	return respondAutocomplete(session, interaction, choices)
}

func respondAutocomplete(session pollSession, interaction *discordgo.Interaction, choices []*discordgo.ApplicationCommandOptionChoice) error {
	return session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}
