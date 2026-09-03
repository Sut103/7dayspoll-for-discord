package poll

import "github.com/bwmarrin/discordgo"

// fakePollSession is a hand-written test double for pollSession.
type fakePollSession struct {
	calls []string

	interactionRespondErr error
	interactionRespondArg *discordgo.InteractionResponse

	interactionResponseResult *discordgo.Message
	interactionResponseErr    error

	guildScheduledEventCreateResult  *discordgo.GuildScheduledEvent
	guildScheduledEventCreateErr     error
	guildScheduledEventCreateGuildID string
	guildScheduledEventCreateParams  *discordgo.GuildScheduledEventParams

	followupMessageCreateResult *discordgo.Message
	followupMessageCreateErr    error
	followupMessageCreateData   *discordgo.WebhookParams
}

func (f *fakePollSession) InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
	f.calls = append(f.calls, "InteractionRespond")
	f.interactionRespondArg = resp
	return f.interactionRespondErr
}

func (f *fakePollSession) InteractionResponse(interaction *discordgo.Interaction, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.calls = append(f.calls, "InteractionResponse")
	if f.interactionResponseErr != nil {
		return nil, f.interactionResponseErr
	}
	return f.interactionResponseResult, nil
}

func (f *fakePollSession) FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.calls = append(f.calls, "FollowupMessageCreate")
	f.followupMessageCreateData = data
	if f.followupMessageCreateErr != nil {
		return nil, f.followupMessageCreateErr
	}
	return f.followupMessageCreateResult, nil
}

func (f *fakePollSession) GuildScheduledEventCreate(guildID string, event *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error) {
	f.calls = append(f.calls, "GuildScheduledEventCreate")
	f.guildScheduledEventCreateGuildID = guildID
	f.guildScheduledEventCreateParams = event
	if f.guildScheduledEventCreateErr != nil {
		return nil, f.guildScheduledEventCreateErr
	}
	return f.guildScheduledEventCreateResult, nil
}

// compile-time check: fakePollSession must satisfy pollSession.
var _ pollSession = (*fakePollSession)(nil)
