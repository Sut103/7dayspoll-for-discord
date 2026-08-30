package manage

import "github.com/bwmarrin/discordgo"

type createCall struct {
	appID   string
	guildID string
	cmd     *discordgo.ApplicationCommand
}

type deleteCall struct {
	appID   string
	guildID string
	cmdID   string
}

// fakeCommandSession is a hand-written test double for commandSession.
type fakeCommandSession struct {
	createCalls []createCall
	// createErrs[i] is returned by the i-th ApplicationCommandCreate call; missing/nil means no error.
	createErrs []error

	applicationCommandsResult []*discordgo.ApplicationCommand
	applicationCommandsErr    error

	deleteCalls []deleteCall
	// deleteErrs[i] is returned by the i-th ApplicationCommandDelete call; missing/nil means no error.
	deleteErrs []error
}

func (f *fakeCommandSession) ApplicationCommandCreate(appID string, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	callIndex := len(f.createCalls)
	f.createCalls = append(f.createCalls, createCall{appID: appID, guildID: guildID, cmd: cmd})
	if callIndex < len(f.createErrs) && f.createErrs[callIndex] != nil {
		return nil, f.createErrs[callIndex]
	}
	return cmd, nil
}

func (f *fakeCommandSession) ApplicationCommands(appID string, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
	if f.applicationCommandsErr != nil {
		return nil, f.applicationCommandsErr
	}
	return f.applicationCommandsResult, nil
}

func (f *fakeCommandSession) ApplicationCommandDelete(appID string, guildID string, cmdID string, options ...discordgo.RequestOption) error {
	callIndex := len(f.deleteCalls)
	f.deleteCalls = append(f.deleteCalls, deleteCall{appID: appID, guildID: guildID, cmdID: cmdID})
	if callIndex < len(f.deleteErrs) && f.deleteErrs[callIndex] != nil {
		return f.deleteErrs[callIndex]
	}
	return nil
}
