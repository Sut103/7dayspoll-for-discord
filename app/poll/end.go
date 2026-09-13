package poll

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func pollCreatorID(message *discordgo.Message) string {
	// message.Author is the bot, not the human who ran /poll.
	if message.InteractionMetadata != nil && message.InteractionMetadata.User != nil {
		return message.InteractionMetadata.User.ID
	}
	return ""
}

func userID(interaction *discordgo.Interaction) string {
	if interaction.Member != nil && interaction.Member.User != nil {
		return interaction.Member.User.ID
	}
	// Member is nil in DMs, where polls can also be created.
	if interaction.User != nil {
		return interaction.User.ID
	}
	return ""
}

func hasManageMessages(interaction *discordgo.Interaction) bool {
	return interaction.Member != nil && interaction.Member.Permissions&discordgo.PermissionManageMessages != 0
}

// Discord only blocks ending another application's poll, so who may end ours is up to us.
func canEndPoll(interaction *discordgo.Interaction, message *discordgo.Message) bool {
	userID := userID(interaction)
	if userID == "" {
		return false
	}
	if id := pollCreatorID(message); id != "" && id == userID {
		return true
	}
	return hasManageMessages(interaction)
}

func isFinalized(poll *discordgo.Poll) bool {
	if poll == nil {
		return false
	}
	if poll.Results != nil && poll.Results.Finalized {
		return true
	}
	// Results may be null even when fetching.
	return poll.Expiry != nil && time.Now().After(*poll.Expiry)
}

var (
	messageIDPattern   = regexp.MustCompile(`^\d{17,20}$`)
	messageLinkPattern = regexp.MustCompile(`^https?://(?:canary\.|ptb\.)?(?:discord|discordapp)\.com/channels/(?:\d+|@me)/(\d+)/(\d+)$`)

	errInvalidMessageReference = errors.New("invalid poll message reference")
)

func resolveMessageID(input, currentChannelID string) (string, error) {
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
