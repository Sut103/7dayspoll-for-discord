package poll

import (
	"errors"
	"log"

	"github.com/bwmarrin/discordgo"
)

func EndMessageCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Type: discordgo.MessageApplicationCommand,
		Name: "End Poll",
		NameLocalizations: &map[discordgo.Locale]string{
			discordgo.Japanese: "投票を終了",
		},
	}
}

func EndSlashCommand() *discordgo.ApplicationCommand {
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

func EndFromMessage(session pollSession, interaction *discordgo.Interaction) error {
	i18n := GetI18n(interaction.Locale)
	data := interaction.ApplicationCommandData()

	var message *discordgo.Message
	if data.Resolved != nil {
		message = data.Resolved.Messages[data.TargetID]
	}
	if message == nil {
		return respondEphemeral(session, interaction, i18n.PollEndTargetNotFound)
	}
	return endPoll(session, interaction, message, i18n)
}

func EndFromSlash(session pollSession, interaction *discordgo.Interaction) error {
	i18n := GetI18n(interaction.Locale)
	options := interaction.ApplicationCommandData().Options
	if len(options) == 0 {
		return respondEphemeral(session, interaction, i18n.PollEndInvalidMessage)
	}

	messageID, err := resolveMessageID(options[0].StringValue(), interaction.ChannelID)
	if err != nil {
		return respondEphemeral(session, interaction, i18n.PollEndInvalidMessage)
	}
	// discordgo returns (nil, nil) when the response body is null.
	message, err := session.ChannelMessage(interaction.ChannelID, messageID)
	if err != nil || message == nil {
		return respondEphemeral(session, interaction, i18n.PollEndTargetNotFound)
	}
	return endPoll(session, interaction, message, i18n)
}

func endPoll(session pollSession, interaction *discordgo.Interaction, message *discordgo.Message, i18n I18n) error {
	// Checked before the poll's own state so a member without rights learns nothing about it.
	if !canEndPoll(interaction, message) {
		return respondEphemeral(session, interaction, i18n.PollEndNoPermission)
	}
	if message.Poll == nil {
		return respondEphemeral(session, interaction, i18n.PollNotFound)
	}
	if isFinalized(message.Poll) {
		return respondEphemeral(session, interaction, i18n.PollAlreadyEnded)
	}

	// PollExpire can outlast Discord's 3-second response window, so acknowledge first.
	if err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		log.Println("Failed to defer poll-end interaction:", err)
		return err
	}

	if _, err := session.PollExpire(message.ChannelID, message.ID); err != nil {
		log.Println("Failed to expire poll:", err)
		// The poll can end between the check above and this call; retrying would never help.
		if isPollExpiredError(err) {
			return editDeferredResponse(session, interaction, i18n.PollAlreadyEnded)
		}
		return editDeferredResponse(session, interaction, i18n.PollEndFailed)
	}
	return editDeferredResponse(session, interaction, i18n.PollEndSuccess)
}

// Discord's JSON error code for an already-expired poll.
const pollExpiredErrorCode = 520001

func isPollExpiredError(err error) bool {
	var restErr *discordgo.RESTError
	if !errors.As(err, &restErr) {
		return false
	}
	return restErr.Message != nil && restErr.Message.Code == pollExpiredErrorCode
}

func respondEphemeral(session pollSession, interaction *discordgo.Interaction, content string) error {
	return session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func editDeferredResponse(session pollSession, interaction *discordgo.Interaction, content string) error {
	_, err := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{Content: &content})
	return err
}
