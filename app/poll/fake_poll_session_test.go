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

	interactionResponseEditResult *discordgo.Message
	interactionResponseEditErr    error
	interactionResponseEditData   *discordgo.WebhookEdit

	channelMessageResult    *discordgo.Message
	channelMessageErr       error
	channelMessageChannelID string
	channelMessageMessageID string

	channelMessagesResult    []*discordgo.Message
	channelMessagesErr       error
	channelMessagesChannelID string
	channelMessagesLimit     int

	pollExpireResult    *discordgo.Message
	pollExpireErr       error
	pollExpireChannelID string
	pollExpireMessageID string
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

func (f *fakePollSession) InteractionResponseEdit(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.calls = append(f.calls, "InteractionResponseEdit")
	f.interactionResponseEditData = newresp
	if f.interactionResponseEditErr != nil {
		return nil, f.interactionResponseEditErr
	}
	return f.interactionResponseEditResult, nil
}

func (f *fakePollSession) ChannelMessage(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.calls = append(f.calls, "ChannelMessage")
	f.channelMessageChannelID = channelID
	f.channelMessageMessageID = messageID
	if f.channelMessageErr != nil {
		return nil, f.channelMessageErr
	}
	return f.channelMessageResult, nil
}

func (f *fakePollSession) ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string, options ...discordgo.RequestOption) ([]*discordgo.Message, error) {
	f.calls = append(f.calls, "ChannelMessages")
	f.channelMessagesChannelID = channelID
	f.channelMessagesLimit = limit
	if f.channelMessagesErr != nil {
		return nil, f.channelMessagesErr
	}
	return f.channelMessagesResult, nil
}

func (f *fakePollSession) PollExpire(channelID, messageID string) (*discordgo.Message, error) {
	f.calls = append(f.calls, "PollExpire")
	f.pollExpireChannelID = channelID
	f.pollExpireMessageID = messageID
	if f.pollExpireErr != nil {
		return nil, f.pollExpireErr
	}
	return f.pollExpireResult, nil
}

// compile-time check: fakePollSession must satisfy pollSession.
var _ pollSession = (*fakePollSession)(nil)
