package manage

import (
	"errors"
	"testing"

	"7DaysPoll/poll"

	"github.com/bwmarrin/discordgo"
)

func TestCommandsToRegister(t *testing.T) {
	commands := commandsToRegister()
	if len(commands) != 2 {
		t.Fatalf("commandsToRegister() returned %d commands, want 2", len(commands))
	}
	wantNames := []string{poll.GetNativePollCommand().Name, poll.GetClassicPollCommand().Name}
	for i, want := range wantNames {
		if commands[i].Name != want {
			t.Errorf("commands[%d].Name = %q, want %q", i, commands[i].Name, want)
		}
	}
}

// fakeCommandSession is a hand-written test double for commandSession: it
// records every call it receives and returns configurable results, so
// Register/Delete's orchestration can be verified without talking to
// Discord.
type fakeCommandSession struct {
	createCalls []createCall
	createErr   error

	commands    []*discordgo.ApplicationCommand
	commandsErr error

	deleteCalls []string
	deleteErr   error
}

type createCall struct {
	appID   string
	guildID string
	name    string
}

func (f *fakeCommandSession) ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, _ ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	f.createCalls = append(f.createCalls, createCall{appID, guildID, cmd.Name})
	if f.createErr != nil {
		return nil, f.createErr
	}
	return cmd, nil
}

func (f *fakeCommandSession) ApplicationCommands(_, _ string, _ ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
	if f.commandsErr != nil {
		return nil, f.commandsErr
	}
	return f.commands, nil
}

func (f *fakeCommandSession) ApplicationCommandDelete(_, _, cmdID string, _ ...discordgo.RequestOption) error {
	f.deleteCalls = append(f.deleteCalls, cmdID)
	return f.deleteErr
}

func TestRegister(t *testing.T) {
	t.Run("creates every command to register with the given appID", func(t *testing.T) {
		fake := &fakeCommandSession{}

		if err := Register(fake, "app-1"); err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		want := commandsToRegister()
		if len(fake.createCalls) != len(want) {
			t.Fatalf("ApplicationCommandCreate called %d times, want %d", len(fake.createCalls), len(want))
		}
		for i, call := range fake.createCalls {
			if call.appID != "app-1" {
				t.Errorf("call[%d].appID = %q, want %q", i, call.appID, "app-1")
			}
			if call.guildID != "" {
				t.Errorf("call[%d].guildID = %q, want empty (global command)", i, call.guildID)
			}
			if call.name != want[i].Name {
				t.Errorf("call[%d].name = %q, want %q", i, call.name, want[i].Name)
			}
		}
	})

	t.Run("stops and returns the error as soon as one command fails to create", func(t *testing.T) {
		wantErr := errors.New("create failed")
		fake := &fakeCommandSession{createErr: wantErr}

		err := Register(fake, "app-1")
		if err != wantErr {
			t.Fatalf("Register() error = %v, want %v", err, wantErr)
		}
		if len(fake.createCalls) != 1 {
			t.Errorf("ApplicationCommandCreate called %d times, want 1 (should stop after the first failure)", len(fake.createCalls))
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("deletes every command returned by ApplicationCommands", func(t *testing.T) {
		fake := &fakeCommandSession{
			commands: []*discordgo.ApplicationCommand{{ID: "cmd-1"}, {ID: "cmd-2"}},
		}

		if err := Delete(fake, "app-1"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if len(fake.deleteCalls) != 2 || fake.deleteCalls[0] != "cmd-1" || fake.deleteCalls[1] != "cmd-2" {
			t.Errorf("deleteCalls = %v, want [cmd-1 cmd-2]", fake.deleteCalls)
		}
	})

	t.Run("no commands registered: returns nil without deleting anything", func(t *testing.T) {
		fake := &fakeCommandSession{commands: nil}

		if err := Delete(fake, "app-1"); err != nil {
			t.Fatalf("Delete() error = %v, want nil", err)
		}
		if len(fake.deleteCalls) != 0 {
			t.Errorf("ApplicationCommandDelete called %d times, want 0", len(fake.deleteCalls))
		}
	})

	t.Run("propagates an error from ApplicationCommands without deleting anything", func(t *testing.T) {
		wantErr := errors.New("list failed")
		fake := &fakeCommandSession{commandsErr: wantErr}

		err := Delete(fake, "app-1")
		if err != wantErr {
			t.Fatalf("Delete() error = %v, want %v", err, wantErr)
		}
		if len(fake.deleteCalls) != 0 {
			t.Errorf("ApplicationCommandDelete called %d times, want 0", len(fake.deleteCalls))
		}
	})

	t.Run("stops and returns the error as soon as one command fails to delete", func(t *testing.T) {
		wantErr := errors.New("delete failed")
		fake := &fakeCommandSession{
			commands:  []*discordgo.ApplicationCommand{{ID: "cmd-1"}, {ID: "cmd-2"}},
			deleteErr: wantErr,
		}

		err := Delete(fake, "app-1")
		if err != wantErr {
			t.Fatalf("Delete() error = %v, want %v", err, wantErr)
		}
		if len(fake.deleteCalls) != 1 {
			t.Errorf("ApplicationCommandDelete called %d times, want 1 (should stop after the first failure)", len(fake.deleteCalls))
		}
	})
}
