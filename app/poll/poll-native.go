package poll

import (
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	defaultDurationDays = 3
	minDurationDays     = 1
	// Discord allows polls to stay open for up to 32 days (768 hours).
	maxDurationDays = 32
	// Discord's poll question text is capped at 300 characters.
	pollQuestionMaxLength = 300
)

func GetNativePollCommand() *discordgo.ApplicationCommand {
	minLength := 5
	minDays := 2
	maxDays := 7
	return &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        "poll",
		Description: "Starting Poll from initial date with specified number of days (2-7).",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "title",
				Description: "Please enter the title of the poll.",
				Type:        discordgo.ApplicationCommandOptionString,
				MaxLength:   pollQuestionMaxLength,
			},
			{
				Name:        "start-date",
				Description: "If you have desired options, please specify the initial date. Example: 08/31",
				Type:        discordgo.ApplicationCommandOptionString,
				MaxLength:   5,
				MinLength:   &minLength,
			},
			{
				Name:        "days",
				Description: "Number of days for the poll (2-7). Default is 7.",
				Type:        discordgo.ApplicationCommandOptionInteger,
				MinValue:    FloatPtr(float64(minDays)),
				MaxValue:    float64(maxDays),
			},
			{
				Name:        "duration",
				Description: "Poll duration in days (1-32). Default is 3.",
				Type:        discordgo.ApplicationCommandOptionInteger,
				MinValue:    FloatPtr(float64(minDurationDays)),
				MaxValue:    float64(maxDurationDays),
			},
		},
	}
}

func NativePoll(session pollSession, interaction *discordgo.Interaction) error {
	i18n := GetI18n(interaction.Locale)
	opts, err := parsePollOptions(interaction, i18n)
	if err != nil {
		return err
	}
	durationDays := defaultDurationDays
	if d, ok := opts.OptMap["duration"]; ok {
		durationDays = int(d.IntValue())
		if durationDays < minDurationDays {
			durationDays = minDurationDays
		} else if durationDays > maxDurationDays {
			durationDays = maxDurationDays
		}
	}
	choices := getChoices(i18n, opts.Start, opts.NumDays)
	answers := make([]discordgo.PollAnswer, 0, len(choices))
	for _, choice := range choices {
		answers = append(answers, discordgo.PollAnswer{
			Media: &discordgo.PollMedia{
				Text:  choice.Name,
				Emoji: &discordgo.ComponentEmoji{Name: choice.Emoji},
			},
		})
	}
	durationHours := durationDays * 24
	// A poll shouldn't outlive the scheduled event linked to it, so clamp the
	// duration to the time remaining before the event's start (only relevant
	// when an event will actually be created, i.e. not in a DM). eventStart is
	// reused below when actually creating the event, so the two can never
	// disagree on when the event starts.
	now := time.Now()
	var eventStart time.Time
	if interaction.GuildID != "" {
		eventStart = resolveEventStartTime(opts.Start, opts.NumDays, now)
		durationHours = clampPollDurationToEvent(durationHours, eventStart, now)
	}
	body := discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Poll: &discordgo.Poll{
				Question:         discordgo.PollMedia{Text: truncateRunes(opts.Title, pollQuestionMaxLength)},
				Answers:          answers,
				AllowMultiselect: true,
				Duration:         durationHours,
			},
		},
	}
	err = session.InteractionRespond(interaction, &body)
	if err != nil {
		log.Println(err)
		return err
	}

	// Guild scheduled events cannot be created from DMs; GuildID is empty in that case.
	if interaction.GuildID == "" {
		return nil
	}

	message, err := session.InteractionResponse(interaction)
	if err != nil {
		return err
	}
	messageURL := buildMessageURL(interaction.GuildID, interaction.ChannelID, message.ID)

	event, err := createScheduledEvent(session, interaction.GuildID, i18n, opts.Start, opts.NumDays, opts.Title, messageURL, eventStart)
	if err != nil {
		log.Println("Failed to create guild scheduled event:", err)
		return nil
	}

	// A message containing a poll cannot be edited, so the event link is posted
	// as a follow-up message instead of being embedded into the poll message.
	_, err = session.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
		Content: buildEventURL(interaction.GuildID, event.ID),
	})
	if err != nil {
		log.Println("Failed to post event link follow-up message:", err)
	}

	return nil
}
