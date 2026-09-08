package poll

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
)

const (
	testPollChannelID = "111111111111111111"
	testPollMessageID = "222222222222222222"
	testCreatorID     = "creator-1"
)

func newPollMessage() *discordgo.Message {
	return &discordgo.Message{
		ID:                  testPollMessageID,
		ChannelID:           testPollChannelID,
		Poll:                &discordgo.Poll{Question: discordgo.PollMedia{Text: "予定調整"}},
		InteractionMetadata: &discordgo.MessageInteractionMetadata{User: &discordgo.User{ID: testCreatorID}},
	}
}

func newCreatorInteraction() *discordgo.Interaction {
	return &discordgo.Interaction{
		Type:      discordgo.InteractionApplicationCommand,
		ChannelID: testPollChannelID,
		Locale:    discordgo.EnglishUS,
		Member:    &discordgo.Member{User: &discordgo.User{ID: testCreatorID}},
	}
}

func newBystanderInteraction() *discordgo.Interaction {
	return &discordgo.Interaction{
		Type:      discordgo.InteractionApplicationCommand,
		ChannelID: testPollChannelID,
		Locale:    discordgo.EnglishUS,
		Member:    &discordgo.Member{User: &discordgo.User{ID: "bystander-1"}},
	}
}

func newModeratorInteraction() *discordgo.Interaction {
	return &discordgo.Interaction{
		Type:      discordgo.InteractionApplicationCommand,
		ChannelID: testPollChannelID,
		Locale:    discordgo.EnglishUS,
		Member: &discordgo.Member{
			User:        &discordgo.User{ID: "moderator-1"},
			Permissions: discordgo.PermissionManageMessages,
		},
	}
}

func withMessageCommandData(interaction *discordgo.Interaction, message *discordgo.Message) *discordgo.Interaction {
	data := discordgo.ApplicationCommandInteractionData{
		Name:        "End Poll",
		CommandType: discordgo.MessageApplicationCommand,
	}
	if message != nil {
		data.TargetID = message.ID
		data.Resolved = &discordgo.ApplicationCommandInteractionDataResolved{
			Messages: map[string]*discordgo.Message{message.ID: message},
		}
	}
	interaction.Data = data
	return interaction
}

func withSlashCommandData(interaction *discordgo.Interaction, messageOption string) *discordgo.Interaction {
	interaction.Data = discordgo.ApplicationCommandInteractionData{
		Name:        "poll-end",
		CommandType: discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandInteractionDataOption{
			{Name: "message", Type: discordgo.ApplicationCommandOptionString, Value: messageOption},
		},
	}
	return interaction
}

func respondedContent(t *testing.T, fake *fakePollSession) string {
	t.Helper()
	if fake.interactionRespondArg == nil || fake.interactionRespondArg.Data == nil {
		t.Fatalf("no interaction response was sent")
	}
	return fake.interactionRespondArg.Data.Content
}

func editedContent(t *testing.T, fake *fakePollSession) string {
	t.Helper()
	if fake.interactionResponseEditData == nil || fake.interactionResponseEditData.Content == nil {
		t.Fatalf("no deferred response was edited")
	}
	return *fake.interactionResponseEditData.Content
}

func TestEndMessageCommand(t *testing.T) {
	command := EndMessageCommand()

	if command.Type != discordgo.MessageApplicationCommand {
		t.Errorf("Type = %v, want %v", command.Type, discordgo.MessageApplicationCommand)
	}
	if command.Name != "End Poll" {
		t.Errorf("Name = %q, want %q", command.Name, "End Poll")
	}
	if command.NameLocalizations == nil {
		t.Fatalf("NameLocalizations is nil, want a Japanese name")
	}
	if got := (*command.NameLocalizations)[discordgo.Japanese]; got != "投票を終了" {
		t.Errorf("Japanese name = %q, want %q", got, "投票を終了")
	}
}

func TestEndSlashCommand(t *testing.T) {
	command := EndSlashCommand()

	if command.Type != discordgo.ChatApplicationCommand {
		t.Errorf("Type = %v, want %v", command.Type, discordgo.ChatApplicationCommand)
	}
	if command.Name != "poll-end" {
		t.Errorf("Name = %q, want %q", command.Name, "poll-end")
	}
	if len(command.Options) != 1 {
		t.Fatalf("len(Options) = %d, want 1", len(command.Options))
	}
	option := command.Options[0]
	if option.Name != "message" {
		t.Errorf("option name = %q, want %q", option.Name, "message")
	}
	if option.Type != discordgo.ApplicationCommandOptionString {
		t.Errorf("option type = %v, want %v", option.Type, discordgo.ApplicationCommandOptionString)
	}
	if !option.Required {
		t.Error("option Required = false, want true")
	}
	if !option.Autocomplete {
		t.Error("option Autocomplete = false, want true")
	}
}

func TestEndPoll_NoPermission(t *testing.T) {
	fake := &fakePollSession{}
	i18n := GetI18n(discordgo.EnglishUS)

	if err := endPoll(fake, newBystanderInteraction(), newPollMessage(), i18n); err != nil {
		t.Fatalf("endPoll returned unexpected error: %v", err)
	}

	wantCalls := []string{"InteractionRespond"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v (a member without rights must not reach PollExpire)", fake.calls, wantCalls)
	}
	if got := respondedContent(t, fake); got != i18n.PollEndNoPermission {
		t.Errorf("content = %q, want %q", got, i18n.PollEndNoPermission)
	}
	if fake.interactionRespondArg.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Error("response is not ephemeral")
	}
}

// Identical responses keep a member without rights from probing the message's state.
func TestEndPoll_NoPermissionHidesPollState(t *testing.T) {
	i18n := GetI18n(discordgo.EnglishUS)

	messageWithoutPoll := newPollMessage()
	messageWithoutPoll.Poll = nil
	endedMessage := newPollMessage()
	endedMessage.Poll.Results = &discordgo.PollResults{Finalized: true}

	for _, message := range []*discordgo.Message{newPollMessage(), messageWithoutPoll, endedMessage} {
		fake := &fakePollSession{}
		if err := endPoll(fake, newBystanderInteraction(), message, i18n); err != nil {
			t.Fatalf("endPoll returned unexpected error: %v", err)
		}
		if got := respondedContent(t, fake); got != i18n.PollEndNoPermission {
			t.Errorf("content = %q, want %q for every poll state", got, i18n.PollEndNoPermission)
		}
	}
}

func TestEndPoll_PollNotFound(t *testing.T) {
	fake := &fakePollSession{}
	i18n := GetI18n(discordgo.EnglishUS)
	message := newPollMessage()
	message.Poll = nil

	if err := endPoll(fake, newCreatorInteraction(), message, i18n); err != nil {
		t.Fatalf("endPoll returned unexpected error: %v", err)
	}

	wantCalls := []string{"InteractionRespond"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
	if got := respondedContent(t, fake); got != i18n.PollNotFound {
		t.Errorf("content = %q, want %q", got, i18n.PollNotFound)
	}
}

func TestEndPoll_AlreadyEnded(t *testing.T) {
	fake := &fakePollSession{}
	i18n := GetI18n(discordgo.EnglishUS)
	message := newPollMessage()
	message.Poll.Results = &discordgo.PollResults{Finalized: true}

	if err := endPoll(fake, newCreatorInteraction(), message, i18n); err != nil {
		t.Fatalf("endPoll returned unexpected error: %v", err)
	}

	wantCalls := []string{"InteractionRespond"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
	if got := respondedContent(t, fake); got != i18n.PollAlreadyEnded {
		t.Errorf("content = %q, want %q", got, i18n.PollAlreadyEnded)
	}
}

func TestEndPoll_Success(t *testing.T) {
	fake := &fakePollSession{pollExpireResult: newPollMessage()}
	i18n := GetI18n(discordgo.EnglishUS)

	if err := endPoll(fake, newCreatorInteraction(), newPollMessage(), i18n); err != nil {
		t.Fatalf("endPoll returned unexpected error: %v", err)
	}

	wantCalls := []string{"InteractionRespond", "PollExpire", "InteractionResponseEdit"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
	if fake.interactionRespondArg.Type != discordgo.InteractionResponseDeferredChannelMessageWithSource {
		t.Errorf("response type = %v, want a deferred response", fake.interactionRespondArg.Type)
	}
	if fake.pollExpireChannelID != testPollChannelID || fake.pollExpireMessageID != testPollMessageID {
		t.Errorf("PollExpire(%q, %q), want (%q, %q)", fake.pollExpireChannelID, fake.pollExpireMessageID, testPollChannelID, testPollMessageID)
	}
	if got := editedContent(t, fake); got != i18n.PollEndSuccess {
		t.Errorf("content = %q, want %q", got, i18n.PollEndSuccess)
	}
}

func TestEndPoll_ModeratorCanEndSomeoneElsesPoll(t *testing.T) {
	fake := &fakePollSession{pollExpireResult: newPollMessage()}
	i18n := GetI18n(discordgo.EnglishUS)

	if err := endPoll(fake, newModeratorInteraction(), newPollMessage(), i18n); err != nil {
		t.Fatalf("endPoll returned unexpected error: %v", err)
	}

	wantCalls := []string{"InteractionRespond", "PollExpire", "InteractionResponseEdit"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
	if got := editedContent(t, fake); got != i18n.PollEndSuccess {
		t.Errorf("content = %q, want %q", got, i18n.PollEndSuccess)
	}
}

func TestEndPoll_PollExpireError(t *testing.T) {
	fake := &fakePollSession{pollExpireErr: errors.New("expire failed")}
	i18n := GetI18n(discordgo.EnglishUS)

	if err := endPoll(fake, newCreatorInteraction(), newPollMessage(), i18n); err != nil {
		t.Fatalf("endPoll returned unexpected error: %v", err)
	}

	wantCalls := []string{"InteractionRespond", "PollExpire", "InteractionResponseEdit"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
	if got := editedContent(t, fake); got != i18n.PollEndFailed {
		t.Errorf("content = %q, want %q", got, i18n.PollEndFailed)
	}
}

func TestEndPoll_DeferError(t *testing.T) {
	wantErr := errors.New("defer failed")
	fake := &fakePollSession{interactionRespondErr: wantErr}
	i18n := GetI18n(discordgo.EnglishUS)

	err := endPoll(fake, newCreatorInteraction(), newPollMessage(), i18n)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}

	wantCalls := []string{"InteractionRespond"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v (a failed defer must not expire the poll)", fake.calls, wantCalls)
	}
}

func TestEndFromMessage_Success(t *testing.T) {
	fake := &fakePollSession{pollExpireResult: newPollMessage()}
	i18n := GetI18n(discordgo.EnglishUS)
	interaction := withMessageCommandData(newCreatorInteraction(), newPollMessage())

	if err := EndFromMessage(fake, interaction); err != nil {
		t.Fatalf("EndFromMessage returned unexpected error: %v", err)
	}

	wantCalls := []string{"InteractionRespond", "PollExpire", "InteractionResponseEdit"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
	if got := editedContent(t, fake); got != i18n.PollEndSuccess {
		t.Errorf("content = %q, want %q", got, i18n.PollEndSuccess)
	}
}

// The same answer regardless of rights: the target could not be resolved, so whether
// the caller is the poll's creator is unknowable here.
func TestEndFromMessage_UnresolvedTarget(t *testing.T) {
	i18n := GetI18n(discordgo.EnglishUS)

	for _, interaction := range []*discordgo.Interaction{
		withMessageCommandData(newModeratorInteraction(), nil),
		withMessageCommandData(newCreatorInteraction(), nil),
		withMessageCommandData(newBystanderInteraction(), nil),
	} {
		fake := &fakePollSession{}
		if err := EndFromMessage(fake, interaction); err != nil {
			t.Fatalf("EndFromMessage returned unexpected error: %v", err)
		}
		wantCalls := []string{"InteractionRespond"}
		if !reflect.DeepEqual(fake.calls, wantCalls) {
			t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
		}
		if got := respondedContent(t, fake); got != i18n.PollEndTargetNotFound {
			t.Errorf("content = %q, want %q", got, i18n.PollEndTargetNotFound)
		}
	}
}

func TestEndFromSlash_Success(t *testing.T) {
	message := newPollMessage()
	fake := &fakePollSession{channelMessageResult: message, pollExpireResult: message}
	i18n := GetI18n(discordgo.EnglishUS)
	interaction := withSlashCommandData(newCreatorInteraction(), testPollMessageID)

	if err := EndFromSlash(fake, interaction); err != nil {
		t.Fatalf("EndFromSlash returned unexpected error: %v", err)
	}

	wantCalls := []string{"ChannelMessage", "InteractionRespond", "PollExpire", "InteractionResponseEdit"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
	if fake.channelMessageChannelID != testPollChannelID || fake.channelMessageMessageID != testPollMessageID {
		t.Errorf("ChannelMessage(%q, %q), want (%q, %q)", fake.channelMessageChannelID, fake.channelMessageMessageID, testPollChannelID, testPollMessageID)
	}
	if got := editedContent(t, fake); got != i18n.PollEndSuccess {
		t.Errorf("content = %q, want %q", got, i18n.PollEndSuccess)
	}
}

func TestEndFromSlash_InvalidMessage(t *testing.T) {
	i18n := GetI18n(discordgo.EnglishUS)

	tests := []struct {
		name  string
		input string
	}{
		{"not a message reference at all", "yesterday's poll"},
		{"a link pointing at another channel", "https://discord.com/channels/999999999999999999/333333333333333333/" + testPollMessageID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakePollSession{}
			interaction := withSlashCommandData(newCreatorInteraction(), tt.input)

			if err := EndFromSlash(fake, interaction); err != nil {
				t.Fatalf("EndFromSlash returned unexpected error: %v", err)
			}

			wantCalls := []string{"InteractionRespond"}
			if !reflect.DeepEqual(fake.calls, wantCalls) {
				t.Fatalf("calls = %v, want %v (an unparsable reference must not be fetched)", fake.calls, wantCalls)
			}
			if got := respondedContent(t, fake); got != i18n.PollEndInvalidMessage {
				t.Errorf("content = %q, want %q", got, i18n.PollEndInvalidMessage)
			}
		})
	}
}

// Discord always sends a required option, but a missing one must not take the bot down.
func TestEndFromSlash_MissingOption(t *testing.T) {
	fake := &fakePollSession{}
	i18n := GetI18n(discordgo.EnglishUS)
	interaction := newCreatorInteraction()
	interaction.Data = discordgo.ApplicationCommandInteractionData{
		Name:        "poll-end",
		CommandType: discordgo.ChatApplicationCommand,
	}

	if err := EndFromSlash(fake, interaction); err != nil {
		t.Fatalf("EndFromSlash returned unexpected error: %v", err)
	}

	wantCalls := []string{"InteractionRespond"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
	if got := respondedContent(t, fake); got != i18n.PollEndInvalidMessage {
		t.Errorf("content = %q, want %q", got, i18n.PollEndInvalidMessage)
	}
}

func TestEndFromSlash_ChannelMessageError(t *testing.T) {
	i18n := GetI18n(discordgo.EnglishUS)

	for _, interaction := range []*discordgo.Interaction{
		withSlashCommandData(newModeratorInteraction(), testPollMessageID),
		withSlashCommandData(newCreatorInteraction(), testPollMessageID),
		withSlashCommandData(newBystanderInteraction(), testPollMessageID),
	} {
		fake := &fakePollSession{channelMessageErr: errors.New("not found")}
		if err := EndFromSlash(fake, interaction); err != nil {
			t.Fatalf("EndFromSlash returned unexpected error: %v", err)
		}
		wantCalls := []string{"ChannelMessage", "InteractionRespond"}
		if !reflect.DeepEqual(fake.calls, wantCalls) {
			t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
		}
		if got := respondedContent(t, fake); got != i18n.PollEndTargetNotFound {
			t.Errorf("content = %q, want %q", got, i18n.PollEndTargetNotFound)
		}
	}
}

// discordgo returns (nil, nil) when the body is null, which must not reach endPoll.
func TestEndFromSlash_NilMessageWithoutError(t *testing.T) {
	i18n := GetI18n(discordgo.EnglishUS)
	fake := &fakePollSession{}
	interaction := withSlashCommandData(newCreatorInteraction(), testPollMessageID)

	if err := EndFromSlash(fake, interaction); err != nil {
		t.Fatalf("EndFromSlash returned unexpected error: %v", err)
	}

	wantCalls := []string{"ChannelMessage", "InteractionRespond"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
	if got := respondedContent(t, fake); got != i18n.PollEndTargetNotFound {
		t.Errorf("content = %q, want %q", got, i18n.PollEndTargetNotFound)
	}
}

// The poll can end between the isFinalized check and PollExpire; retrying would never help.
func TestEndPoll_PollExpiredMeanwhile(t *testing.T) {
	i18n := GetI18n(discordgo.EnglishUS)
	fake := &fakePollSession{pollExpireErr: &discordgo.RESTError{
		Message: &discordgo.APIErrorMessage{Code: pollExpiredErrorCode, Message: "Poll expired"},
	}}

	if err := endPoll(fake, newCreatorInteraction(), newPollMessage(), i18n); err != nil {
		t.Fatalf("endPoll returned unexpected error: %v", err)
	}

	if got := editedContent(t, fake); got != i18n.PollAlreadyEnded {
		t.Errorf("content = %q, want %q", got, i18n.PollAlreadyEnded)
	}
}
