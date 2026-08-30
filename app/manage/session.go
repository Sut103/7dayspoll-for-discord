package manage

import "github.com/bwmarrin/discordgo"

// commandSession is the subset of *discordgo.Session's API that this
// package needs in order to register and delete application commands.
// Extracting it as an interface lets Register and Delete be tested
// with a hand-written fake instead of a real Discord session.
type commandSession interface {
	ApplicationCommandCreate(appID string, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	ApplicationCommands(appID string, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error)
	ApplicationCommandDelete(appID string, guildID string, cmdID string, options ...discordgo.RequestOption) error
}
