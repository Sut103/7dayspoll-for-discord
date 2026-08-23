package manage

import "github.com/bwmarrin/discordgo"

// commandSession is the subset of *discordgo.Session's API that Register and
// Delete depend on. Depending on this narrow interface instead of the
// concrete session lets tests substitute a fake implementation instead of
// talking to Discord.
type commandSession interface {
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	ApplicationCommands(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error)
	ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error
}
