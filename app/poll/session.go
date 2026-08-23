package poll

import "github.com/bwmarrin/discordgo"

// pollSession is the subset of *discordgo.Session's API that NativePoll and
// createScheduledEvent depend on. Depending on this narrow interface instead
// of the concrete session lets tests substitute a fake implementation
// instead of talking to Discord.
type pollSession interface {
	InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	InteractionResponse(interaction *discordgo.Interaction, options ...discordgo.RequestOption) (*discordgo.Message, error)
	FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error)
	GuildScheduledEventCreate(guildID string, event *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error)
}
