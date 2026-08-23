package poll

import (
	"errors"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// GetPollEndMessageCommand registers a message context-menu command ("Apps"
// -> right-click a poll message) that lets a member end that poll early.
func GetPollEndMessageCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Type: discordgo.MessageApplicationCommand,
		Name: "End Poll",
		NameLocalizations: &map[discordgo.Locale]string{
			discordgo.Japanese: "投票を終了",
		},
	}
}

// EndPollMessageCommand expires the poll attached to the message the
// context-menu command was invoked on.
func EndPollMessageCommand(session *discordgo.Session, interaction *discordgo.Interaction) error {
	i18n := GetI18n(interaction.Locale)
	data := interaction.ApplicationCommandData()

	var message *discordgo.Message
	if data.Resolved != nil {
		message = data.Resolved.Messages[data.TargetID]
	}
	if message == nil {
		// Folding this into the same generic response canEndPoll would give
		// a non-moderator keeps them from learning anything from the
		// difference between "couldn't resolve the target message" and
		// "resolved, but not yours".
		if hasManageMessages(interaction) {
			return respondEphemeral(session, interaction, i18n.PollNotFound)
		}
		return respondEphemeral(session, interaction, i18n.PollEndNoPermission)
	}
	return endPollMessage(session, interaction, message, i18n)
}

// GetPollEndSlashCommand registers the "/poll-end" slash command, an
// alternative to the "End Poll" message command for users who don't have
// (or don't want to use) the right-click context menu.
func GetPollEndSlashCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        "poll-end",
		Description: "End a poll early. Only works on a poll message in this channel.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "message",
				Description:  "The poll message's link or ID (must be in this channel).",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
				Autocomplete: true,
			},
		},
	}
}

// EndPollSlashCommand is the "/poll-end" handler. It resolves the "message"
// option to a message in the current channel and delegates to the same
// endPollMessage logic the "End Poll" context-menu command uses.
func EndPollSlashCommand(session *discordgo.Session, interaction *discordgo.Interaction) error {
	i18n := GetI18n(interaction.Locale)
	options := interaction.ApplicationCommandData().Options

	messageID, err := resolvePollMessageID(options[0].StringValue(), interaction.ChannelID)
	if err != nil {
		return respondEphemeral(session, interaction, i18n.PollEndInvalidMessage)
	}
	message, err := session.ChannelMessage(interaction.ChannelID, messageID)
	if err != nil {
		// Same fold as EndPollMessageCommand: a non-moderator gets the
		// generic no-permission response instead of learning the message
		// couldn't be found, so the response can't be used to probe whether
		// a given ID is a poll in this channel without the rights
		// endPollMessage would require anyway.
		if hasManageMessages(interaction) {
			return respondEphemeral(session, interaction, i18n.PollNotFound)
		}
		return respondEphemeral(session, interaction, i18n.PollEndNoPermission)
	}
	return endPollMessage(session, interaction, message, i18n)
}

// pollEndAutocompleteScanLimit is how many of the channel's most recent
// messages are scanned for poll-end autocomplete suggestions, in a single
// ChannelMessages call (Discord's per-request maximum).
const pollEndAutocompleteScanLimit = 100

// pollEndAutocompleteCacheTTL bounds how long a channel's poll-message scan
// is reused across autocomplete requests for /poll-end's "message" option,
// since Discord fires a fresh autocomplete interaction on every keystroke
// with no client-side debounce and no way for the bot to reduce that rate.
// This TTL is enough to collapse the burst of requests from one typed word
// into a single REST call; no invalidation beyond the TTL is attempted — a
// poll created or ended during that window simply won't be reflected until
// the entry expires.
const pollEndAutocompleteCacheTTL = 10 * time.Second

// pollEndAutocompleteCacheEvictAfter bounds how long a stale entry can sit
// in pollEndAutocompleteCache before an opportunistic sweep (piggybacked on
// the next cache write) removes it, so a long-running bot queried across
// many channels — including ones later archived or deleted — doesn't grow
// this map without bound.
const pollEndAutocompleteCacheEvictAfter = 10 * pollEndAutocompleteCacheTTL

// pollEndCandidate holds only the fields PollEndAutocomplete needs to build
// a choice, so the cache doesn't retain whole Message objects (content,
// attachments, embeds, reactions, ...) it never looks at again.
type pollEndCandidate struct {
	messageID string
	title     string
}

type pollEndAutocompleteCacheEntry struct {
	candidates []pollEndCandidate
	fetchedAt  time.Time
}

var (
	// pollEndAutocompleteCacheMu guards both maps below.
	pollEndAutocompleteCacheMu sync.Mutex
	pollEndAutocompleteCache   = make(map[string]pollEndAutocompleteCacheEntry)
	// pollEndAutocompleteChannelLocks holds one mutex per channel that has
	// been queried, serializing the check-then-fetch-then-store sequence in
	// pollEndCandidates per channel. Without this, concurrent autocomplete
	// interactions landing within the same keystroke burst (Discord fires
	// one per keystroke, no debounce) could each observe a cache miss and
	// each issue their own ChannelMessages call before either had written
	// its result back, defeating the point of the cache. Unlike the
	// candidate cache, entries here are never evicted: a *sync.Mutex is
	// tiny, and evicting one while another goroutine might be mid-lookup for
	// it would reintroduce the exact race this map exists to prevent.
	pollEndAutocompleteChannelLocks = make(map[string]*sync.Mutex)
)

// pollEndChannelLock returns the per-channel mutex used to serialize cache
// fetches for one channel, creating it on first use.
func pollEndChannelLock(channelID string) *sync.Mutex {
	pollEndAutocompleteCacheMu.Lock()
	defer pollEndAutocompleteCacheMu.Unlock()
	lock, ok := pollEndAutocompleteChannelLocks[channelID]
	if !ok {
		lock = &sync.Mutex{}
		pollEndAutocompleteChannelLocks[channelID] = lock
	}
	return lock
}

// pollEndCandidates returns this bot's own non-finalized polls among the
// channel's recent messages, reusing a cached result for up to
// pollEndAutocompleteCacheTTL. Only this bot's own messages are considered:
// a poll attached by a different app/user would pass this filter but always
// fail canEndPoll's later PollExpire call, since Discord rejects ending a
// poll owned by another application.
func pollEndCandidates(session *discordgo.Session, channelID string) ([]pollEndCandidate, error) {
	lock := pollEndChannelLock(channelID)
	lock.Lock()
	defer lock.Unlock()

	pollEndAutocompleteCacheMu.Lock()
	entry, ok := pollEndAutocompleteCache[channelID]
	pollEndAutocompleteCacheMu.Unlock()
	if ok && time.Since(entry.fetchedAt) < pollEndAutocompleteCacheTTL {
		return entry.candidates, nil
	}

	messages, err := session.ChannelMessages(channelID, pollEndAutocompleteScanLimit, "", "", "")
	if err != nil {
		return nil, err
	}

	var candidates []pollEndCandidate
	for _, message := range messages {
		if message.Poll == nil {
			continue
		}
		if message.Author == nil || message.Author.ID != session.State.User.ID {
			continue
		}
		if pollFinalized(message.Poll) {
			continue
		}
		candidates = append(candidates, pollEndCandidate{
			messageID: message.ID,
			title:     message.Poll.Question.Text,
		})
	}

	now := time.Now()
	pollEndAutocompleteCacheMu.Lock()
	pollEndAutocompleteCache[channelID] = pollEndAutocompleteCacheEntry{candidates: candidates, fetchedAt: now}
	for id, e := range pollEndAutocompleteCache {
		if now.Sub(e.fetchedAt) > pollEndAutocompleteCacheEvictAfter {
			delete(pollEndAutocompleteCache, id)
		}
	}
	pollEndAutocompleteCacheMu.Unlock()
	return candidates, nil
}

// PollEndAutocomplete answers the autocomplete interaction for /poll-end's
// "message" option, matching cached candidate titles against the user's
// partial input and returning up to Discord's 25-choice limit — displaying
// the poll's title while the value sent on selection is the message ID
// (consumed unchanged by resolvePollMessageID).
func PollEndAutocomplete(session *discordgo.Session, interaction *discordgo.Interaction) error {
	data := interaction.ApplicationCommandData()
	query := strings.ToLower(strings.TrimSpace(data.Options[0].StringValue()))

	candidates, err := pollEndCandidates(session, interaction.ChannelID)
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
			Name:  truncateRunes(candidate.title, 100),
			Value: candidate.messageID,
		})
		if len(choices) == 25 {
			break
		}
	}
	return respondAutocomplete(session, interaction, choices)
}

func respondAutocomplete(session *discordgo.Session, interaction *discordgo.Interaction, choices []*discordgo.ApplicationCommandOptionChoice) error {
	return session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

var (
	messageIDPattern   = regexp.MustCompile(`^\d{17,20}$`)
	messageLinkPattern = regexp.MustCompile(`^https?://(?:canary\.|ptb\.)?(?:discord|discordapp)\.com/channels/(?:\d+|@me)/(\d+)/(\d+)$`)

	errInvalidMessageReference = errors.New("invalid poll message reference")
)

// resolvePollMessageID accepts either a bare Discord message ID or a full
// message link (https://discord.com/channels/{guild|@me}/{channel}/{message},
// including canary/ptb/discordapp.com variants) and returns the message ID.
// Links pointing at a channel other than currentChannelID are rejected,
// since canEndPoll's permission check below is only valid for the invoking
// channel (interaction.Member.Permissions is computed for that channel).
func resolvePollMessageID(input, currentChannelID string) (string, error) {
	input = strings.TrimSpace(input)
	if messageIDPattern.MatchString(input) {
		return input, nil
	}
	match := messageLinkPattern.FindStringSubmatch(input)
	if match == nil {
		return "", errInvalidMessageReference
	}
	channelID, messageID := match[1], match[2]
	if channelID != currentChannelID {
		return "", errInvalidMessageReference
	}
	return messageID, nil
}

// endPollMessage contains the guard checks and PollExpire call shared by the
// "End Poll" message command and the "/poll-end" slash command. Only the
// member who originally posted the poll, or a member with "Manage Messages"
// in the channel, may end it early — mirroring who Discord's own UI allows
// to end a native poll. (Discord's REST API only blocks ending a poll owned
// by a *different application*; per-member access control within our own
// bot's polls is entirely our responsibility.)
//
// The permission check runs before the poll-exists/already-ended checks so
// that a member with no rights over the target message learns nothing about
// its state beyond "you can't do that". EndPollMessageCommand and
// EndPollSlashCommand preserve this even for their own message-resolution
// failures (which happen before this function is ever called): a
// non-moderator gets the same PollEndNoPermission response whether the
// message couldn't be resolved or was resolved but isn't theirs to end.
func endPollMessage(session *discordgo.Session, interaction *discordgo.Interaction, message *discordgo.Message, i18n I18n) error {
	if !canEndPoll(interaction, message) {
		return respondEphemeral(session, interaction, i18n.PollEndNoPermission)
	}
	if message.Poll == nil {
		return respondEphemeral(session, interaction, i18n.PollNotFound)
	}
	if pollFinalized(message.Poll) {
		return respondEphemeral(session, interaction, i18n.PollAlreadyEnded)
	}

	// Defer before the PollExpire REST call: if that call's latency
	// approaches Discord's 3-second interaction response window, the
	// eventual response could fail on an expired token even though the
	// poll itself already ended successfully server-side. Deferring
	// immediately acknowledges the interaction; the actual result is
	// delivered via an edit, which has no such deadline.
	if err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		log.Println("Failed to defer poll-end interaction:", err)
		return err
	}

	if _, err := session.PollExpire(message.ChannelID, message.ID); err != nil {
		log.Println("Failed to expire poll:", err)
		return respondDeferredEphemeral(session, interaction, i18n.PollEndFailed)
	}
	return respondDeferredEphemeral(session, interaction, i18n.PollEndSuccess)
}

func respondDeferredEphemeral(session *discordgo.Session, interaction *discordgo.Interaction, content string) error {
	_, err := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{Content: &content})
	return err
}

// pollFinalized reports whether a poll has already ended. Discord's
// Results.Finalized flag is the primary signal, but Results (like Expiry)
// "might be null even when fetching" per Discord's own API docs, so a
// passed Expiry is used as a fallback that needs no extra API call.
func pollFinalized(poll *discordgo.Poll) bool {
	if poll.Results != nil && poll.Results.Finalized {
		return true
	}
	return poll.Expiry != nil && time.Now().After(*poll.Expiry)
}

// pollCreatorID returns the ID of the human who ran the slash command that
// produced this poll message. message.Author is the bot (it sent the
// interaction response), so the real invoker is recorded in
// message.InteractionMetadata.User instead (Message.Interaction is
// deprecated by Discord in favor of InteractionMetadata, so it is not used).
func pollCreatorID(message *discordgo.Message) string {
	if message.InteractionMetadata != nil && message.InteractionMetadata.User != nil {
		return message.InteractionMetadata.User.ID
	}
	return ""
}

// interactionUserID returns the ID of the human who invoked the interaction,
// whether it happened in a guild (interaction.Member) or a DM
// (interaction.Member is nil there; interaction.User is set instead). Native
// polls can be created via /poll in a DM (poll-native.go's NativePoll only
// skips the guild-scheduled-event step there), so /poll-end and "End Poll"
// must recognize the DM invoker too.
func interactionUserID(interaction *discordgo.Interaction) string {
	if interaction.Member != nil && interaction.Member.User != nil {
		return interaction.Member.User.ID
	}
	if interaction.User != nil {
		return interaction.User.ID
	}
	return ""
}

func canEndPoll(interaction *discordgo.Interaction, message *discordgo.Message) bool {
	userID := interactionUserID(interaction)
	if userID == "" {
		return false
	}
	if id := pollCreatorID(message); id != "" && id == userID {
		return true
	}
	return hasManageMessages(interaction)
}

// hasManageMessages reports whether the invoking member has the "Manage
// Messages" permission in the channel the interaction was sent in. Unlike
// canEndPoll, this doesn't need a specific message, so callers can check it
// before a message has even been fetched. It's always false in a DM
// (interaction.Member is nil there): "Manage Messages" is a per-channel
// guild permission with no meaning outside a guild channel.
func hasManageMessages(interaction *discordgo.Interaction) bool {
	return interaction.Member != nil && interaction.Member.Permissions&discordgo.PermissionManageMessages != 0
}

func respondEphemeral(session *discordgo.Session, interaction *discordgo.Interaction, content string) error {
	return session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
