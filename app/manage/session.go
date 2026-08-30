package manage

import "github.com/bwmarrin/discordgo"

// commandSession narrows *discordgo.Session so Register/Delete can be tested with a fake.
type commandSession interface {
	ApplicationCommandCreate(appID string, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	ApplicationCommands(appID string, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error)
	ApplicationCommandDelete(appID string, guildID string, cmdID string, options ...discordgo.RequestOption) error
}
