package manage

import "github.com/bwmarrin/discordgo"

// createCall records a single ApplicationCommandCreate invocation.
type createCall struct {
	appID   string
	guildID string
	cmd     *discordgo.ApplicationCommand
}

// deleteCall records a single ApplicationCommandDelete invocation.
type deleteCall struct {
	appID   string
	guildID string
	cmdID   string
}

// fakeCommandSession is a hand-written fake implementation of
// commandSession that records call order/arguments and lets tests
// inject return values/errors per call, without relying on a mock
// library.
type fakeCommandSession struct {
	// createCalls records every ApplicationCommandCreate call, in order.
	createCalls []createCall
	// createErrs, indexed by call number (0-based), is returned by the
	// corresponding ApplicationCommandCreate call. A missing entry (or nil)
	// means no error.
	createErrs []error

	// applicationCommandsResult/applicationCommandsErr are returned by
	// ApplicationCommands.
	applicationCommandsResult []*discordgo.ApplicationCommand
	applicationCommandsErr    error

	// deleteCalls records every ApplicationCommandDelete call, in order.
	deleteCalls []deleteCall
	// deleteErrs, indexed by call number (0-based), is returned by the
	// corresponding ApplicationCommandDelete call. A missing entry (or nil)
	// means no error.
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
