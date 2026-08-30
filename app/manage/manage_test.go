package manage

import (
	"errors"
	"testing"

	"7dayspoll/poll"

	"github.com/bwmarrin/discordgo"
)

const testAppID = "test-app-id"

func TestRegister_CreatesNativeThenClassicCommands(t *testing.T) {
	fake := &fakeCommandSession{}

	err := Register(fake, testAppID)
	if err != nil {
		t.Fatalf("Register() returned unexpected error: %v", err)
	}

	if len(fake.createCalls) != 2 {
		t.Fatalf("expected 2 ApplicationCommandCreate calls, got %d", len(fake.createCalls))
	}

	nativeCmd := poll.GetNativePollCommand()
	classicCmd := poll.GetClassicPollCommand()

	first := fake.createCalls[0]
	if first.appID != testAppID {
		t.Errorf("call 1: appID = %q, want %q", first.appID, testAppID)
	}
	if first.guildID != "" {
		t.Errorf("call 1: guildID = %q, want empty string", first.guildID)
	}
	if first.cmd.Name != nativeCmd.Name {
		t.Errorf("call 1: cmd.Name = %q, want %q (native poll command)", first.cmd.Name, nativeCmd.Name)
	}

	second := fake.createCalls[1]
	if second.appID != testAppID {
		t.Errorf("call 2: appID = %q, want %q", second.appID, testAppID)
	}
	if second.guildID != "" {
		t.Errorf("call 2: guildID = %q, want empty string", second.guildID)
	}
	if second.cmd.Name != classicCmd.Name {
		t.Errorf("call 2: cmd.Name = %q, want %q (classic poll command)", second.cmd.Name, classicCmd.Name)
	}
}

func TestRegister_FirstCreateFails_ReturnsErrorAndSkipsSecond(t *testing.T) {
	wantErr := errors.New("native command creation failed")
	fake := &fakeCommandSession{
		createErrs: []error{wantErr},
	}

	err := Register(fake, testAppID)

	if !errors.Is(err, wantErr) {
		t.Fatalf("Register() error = %v, want %v", err, wantErr)
	}
	if len(fake.createCalls) != 1 {
		t.Fatalf("expected 1 ApplicationCommandCreate call (second should be skipped), got %d", len(fake.createCalls))
	}
}

func TestRegister_SecondCreateFails_ReturnsError(t *testing.T) {
	wantErr := errors.New("classic command creation failed")
	fake := &fakeCommandSession{
		createErrs: []error{nil, wantErr},
	}

	err := Register(fake, testAppID)

	if !errors.Is(err, wantErr) {
		t.Fatalf("Register() error = %v, want %v", err, wantErr)
	}
	if len(fake.createCalls) != 2 {
		t.Fatalf("expected 2 ApplicationCommandCreate calls (first already happened), got %d", len(fake.createCalls))
	}
}

func TestDelete_DeletesEveryExistingCommand(t *testing.T) {
	existing := []*discordgo.ApplicationCommand{
		{ID: "cmd-1"},
		{ID: "cmd-2"},
		{ID: "cmd-3"},
	}
	fake := &fakeCommandSession{
		applicationCommandsResult: existing,
	}

	err := Delete(fake, testAppID)
	if err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}

	if len(fake.deleteCalls) != len(existing) {
		t.Fatalf("expected %d ApplicationCommandDelete calls, got %d", len(existing), len(fake.deleteCalls))
	}
	for i, want := range existing {
		got := fake.deleteCalls[i]
		if got.appID != testAppID {
			t.Errorf("delete call %d: appID = %q, want %q", i, got.appID, testAppID)
		}
		if got.guildID != "" {
			t.Errorf("delete call %d: guildID = %q, want empty string", i, got.guildID)
		}
		if got.cmdID != want.ID {
			t.Errorf("delete call %d: cmdID = %q, want %q", i, got.cmdID, want.ID)
		}
	}
}

func TestDelete_NoExistingCommands_ReturnsNilWithoutDeleting(t *testing.T) {
	fake := &fakeCommandSession{
		applicationCommandsResult: []*discordgo.ApplicationCommand{},
	}

	err := Delete(fake, testAppID)
	if err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}
	if len(fake.deleteCalls) != 0 {
		t.Fatalf("expected 0 ApplicationCommandDelete calls, got %d", len(fake.deleteCalls))
	}
}

func TestDelete_ApplicationCommandsErrors_ReturnsErrorWithoutDeleting(t *testing.T) {
	wantErr := errors.New("failed to fetch commands")
	fake := &fakeCommandSession{
		applicationCommandsErr: wantErr,
	}

	err := Delete(fake, testAppID)

	if !errors.Is(err, wantErr) {
		t.Fatalf("Delete() error = %v, want %v", err, wantErr)
	}
	if len(fake.deleteCalls) != 0 {
		t.Fatalf("expected 0 ApplicationCommandDelete calls, got %d", len(fake.deleteCalls))
	}
}

func TestDelete_MidLoopDeleteFails_ReturnsErrorAndStopsRemaining(t *testing.T) {
	existing := []*discordgo.ApplicationCommand{
		{ID: "cmd-1"},
		{ID: "cmd-2"},
		{ID: "cmd-3"},
	}
	wantErr := errors.New("failed to delete cmd-2")
	fake := &fakeCommandSession{
		applicationCommandsResult: existing,
		deleteErrs:                []error{nil, wantErr},
	}

	err := Delete(fake, testAppID)

	if !errors.Is(err, wantErr) {
		t.Fatalf("Delete() error = %v, want %v", err, wantErr)
	}
	if len(fake.deleteCalls) != 2 {
		t.Fatalf("expected 2 ApplicationCommandDelete calls (third should be skipped), got %d", len(fake.deleteCalls))
	}
}
