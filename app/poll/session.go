package poll

import "github.com/bwmarrin/discordgo"

// pollSession is the subset of *discordgo.Session's API that the poll
// package depends on. Narrowing to an interface lets tests substitute a
// hand-written fake instead of a real Discord connection.
type pollSession interface {
	InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	InteractionResponse(interaction *discordgo.Interaction, options ...discordgo.RequestOption) (*discordgo.Message, error)
	FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error)
	GuildScheduledEventCreate(guildID string, event *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error)
}
