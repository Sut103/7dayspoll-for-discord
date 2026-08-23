package poll

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestGetNativePollCommand(t *testing.T) {
	cmd := GetNativePollCommand()

	if cmd.Name != "poll" {
		t.Errorf("Name = %q, want %q", cmd.Name, "poll")
	}
	if cmd.Type != discordgo.ChatApplicationCommand {
		t.Errorf("Type = %v, want %v", cmd.Type, discordgo.ChatApplicationCommand)
	}

	optionsByName := map[string]*discordgo.ApplicationCommandOption{}
	for _, opt := range cmd.Options {
		optionsByName[opt.Name] = opt
	}

	titleOpt, ok := optionsByName["title"]
	if !ok {
		t.Fatal("expected a \"title\" option")
	}
	if titleOpt.Type != discordgo.ApplicationCommandOptionString {
		t.Errorf("title option Type = %v, want string", titleOpt.Type)
	}
	if titleOpt.MaxLength != pollQuestionMaxLength {
		t.Errorf("title option MaxLength = %d, want %d (Discord's poll question limit)", titleOpt.MaxLength, pollQuestionMaxLength)
	}

	daysOpt, ok := optionsByName["days"]
	if !ok {
		t.Fatal("expected a \"days\" option")
	}
	if daysOpt.MinValue == nil || *daysOpt.MinValue != 2 {
		t.Errorf("days option MinValue = %v, want 2", daysOpt.MinValue)
	}
	if daysOpt.MaxValue != 7 {
		t.Errorf("days option MaxValue = %v, want 7", daysOpt.MaxValue)
	}

	durationOpt, ok := optionsByName["duration"]
	if !ok {
		t.Fatal("expected a \"duration\" option")
	}
	if durationOpt.MinValue == nil || *durationOpt.MinValue != minDurationDays {
		t.Errorf("duration option MinValue = %v, want %d", durationOpt.MinValue, minDurationDays)
	}
	if durationOpt.MaxValue != maxDurationDays {
		t.Errorf("duration option MaxValue = %v, want %d", durationOpt.MaxValue, maxDurationDays)
	}

	startDateOpt, ok := optionsByName["start-date"]
	if !ok {
		t.Fatal("expected a \"start-date\" option")
	}
	if startDateOpt.MinLength == nil || *startDateOpt.MinLength != 5 {
		t.Errorf("start-date option MinLength = %v, want 5", startDateOpt.MinLength)
	}
	if startDateOpt.MaxLength != 5 {
		t.Errorf("start-date option MaxLength = %d, want 5", startDateOpt.MaxLength)
	}
}

func TestResolveDurationHours(t *testing.T) {
	tests := []struct {
		name string
		opt  *discordgo.ApplicationCommandInteractionDataOption
		want int
	}{
		{"not given defaults to defaultDurationDays", nil, defaultDurationDays * 24},
		{"below minimum is clamped up to minDurationDays", intOpt("duration", 0), minDurationDays * 24},
		{"above maximum is clamped down to maxDurationDays", intOpt("duration", 100), maxDurationDays * 24},
		{"within range is kept as-is", intOpt("duration", 5), 5 * 24},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optMap := map[string]*discordgo.ApplicationCommandInteractionDataOption{}
			if tt.opt != nil {
				optMap["duration"] = tt.opt
			}
			if got := resolveDurationHours(optMap); got != tt.want {
				t.Errorf("resolveDurationHours() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildPollAnswers(t *testing.T) {
	choices := []Choice{
		{Emoji: "1⃣", Name: "08/21 (Fri)"},
		{Emoji: "❌", Name: "Absence"},
	}
	answers := buildPollAnswers(choices)
	if len(answers) != len(choices) {
		t.Fatalf("buildPollAnswers() returned %d answers, want %d", len(answers), len(choices))
	}
	for i, choice := range choices {
		if answers[i].Media.Text != choice.Name {
			t.Errorf("answers[%d].Media.Text = %q, want %q", i, answers[i].Media.Text, choice.Name)
		}
		if answers[i].Media.Emoji.Name != choice.Emoji {
			t.Errorf("answers[%d].Media.Emoji.Name = %q, want %q", i, answers[i].Media.Emoji.Name, choice.Emoji)
		}
	}
}

func TestBuildNativePollResponse(t *testing.T) {
	answers := []discordgo.PollAnswer{{Media: &discordgo.PollMedia{Text: "08/21 (Fri)"}}}
	longTitle := ""
	for i := 0; i < pollQuestionMaxLength+10; i++ {
		longTitle += "a"
	}

	resp := buildNativePollResponse(longTitle, answers, 72)

	if resp.Type != discordgo.InteractionResponseChannelMessageWithSource {
		t.Errorf("Type = %v, want %v", resp.Type, discordgo.InteractionResponseChannelMessageWithSource)
	}
	poll := resp.Data.Poll
	if poll == nil {
		t.Fatal("Data.Poll is nil")
	}
	if len([]rune(poll.Question.Text)) != pollQuestionMaxLength {
		t.Errorf("Question.Text length = %d, want %d (truncated to Discord's limit)", len([]rune(poll.Question.Text)), pollQuestionMaxLength)
	}
	if !poll.AllowMultiselect {
		t.Error("AllowMultiselect = false, want true")
	}
	if poll.Duration != 72 {
		t.Errorf("Duration = %d, want 72", poll.Duration)
	}
	if len(poll.Answers) != 1 || poll.Answers[0].Media.Text != "08/21 (Fri)" {
		t.Errorf("Answers = %+v, want the given answers to be passed through unchanged", poll.Answers)
	}
}

// fakePollSession is a hand-written test double for pollSession: it records
// every call it receives and returns configurable results, so NativePoll's
// orchestration (which methods are called, in what order, with what data)
// can be verified without talking to Discord.
type fakePollSession struct {
	interactionRespondCalls []*discordgo.InteractionResponse
	interactionRespondErr   error

	interactionResponseCalls   int
	interactionResponseMessage *discordgo.Message
	interactionResponseErr     error

	guildScheduledEventCreateCalls []guildScheduledEventCreateCall
	guildScheduledEventCreateEvent *discordgo.GuildScheduledEvent
	guildScheduledEventCreateErr   error

	followupMessageCreateCalls []*discordgo.WebhookParams
	followupMessageCreateErr   error
}

type guildScheduledEventCreateCall struct {
	guildID string
	params  *discordgo.GuildScheduledEventParams
}

func (f *fakePollSession) InteractionRespond(_ *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
	f.interactionRespondCalls = append(f.interactionRespondCalls, resp)
	return f.interactionRespondErr
}

func (f *fakePollSession) InteractionResponse(_ *discordgo.Interaction, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.interactionResponseCalls++
	if f.interactionResponseErr != nil {
		return nil, f.interactionResponseErr
	}
	if f.interactionResponseMessage != nil {
		return f.interactionResponseMessage, nil
	}
	return &discordgo.Message{ID: "message-1"}, nil
}

func (f *fakePollSession) FollowupMessageCreate(_ *discordgo.Interaction, _ bool, data *discordgo.WebhookParams, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.followupMessageCreateCalls = append(f.followupMessageCreateCalls, data)
	return nil, f.followupMessageCreateErr
}

func (f *fakePollSession) GuildScheduledEventCreate(guildID string, event *discordgo.GuildScheduledEventParams, _ ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error) {
	f.guildScheduledEventCreateCalls = append(f.guildScheduledEventCreateCalls, guildScheduledEventCreateCall{guildID, event})
	if f.guildScheduledEventCreateErr != nil {
		return nil, f.guildScheduledEventCreateErr
	}
	if f.guildScheduledEventCreateEvent != nil {
		return f.guildScheduledEventCreateEvent, nil
	}
	return &discordgo.GuildScheduledEvent{ID: "event-1"}, nil
}

func newGuildInteraction() *discordgo.Interaction {
	interaction := newCommandInteraction(discordgo.EnglishUS, nil)
	interaction.GuildID = "guild-1"
	interaction.ChannelID = "channel-1"
	return interaction
}

func TestNativePoll(t *testing.T) {
	t.Run("in a DM, only responds with the poll and does nothing guild-related", func(t *testing.T) {
		fake := &fakePollSession{}
		interaction := newCommandInteraction(discordgo.EnglishUS, nil) // GuildID is "" by default: a DM

		if err := NativePoll(fake, interaction); err != nil {
			t.Fatalf("NativePoll() error = %v", err)
		}

		if len(fake.interactionRespondCalls) != 1 {
			t.Fatalf("InteractionRespond called %d times, want 1", len(fake.interactionRespondCalls))
		}
		poll := fake.interactionRespondCalls[0].Data.Poll
		if len(poll.Answers) != 8 {
			t.Errorf("Answers count = %d, want 8 (7 default days + absence)", len(poll.Answers))
		}
		if poll.Duration != defaultDurationDays*24 {
			t.Errorf("Duration = %d, want %d", poll.Duration, defaultDurationDays*24)
		}
		if fake.interactionResponseCalls != 0 {
			t.Error("InteractionResponse was called, want it not to be (no scheduled event in a DM)")
		}
		if len(fake.guildScheduledEventCreateCalls) != 0 {
			t.Error("GuildScheduledEventCreate was called, want it not to be (no scheduled event in a DM)")
		}
		if len(fake.followupMessageCreateCalls) != 0 {
			t.Error("FollowupMessageCreate was called, want it not to be (no scheduled event in a DM)")
		}
	})

	t.Run("in a guild, creates the poll, the scheduled event, and the follow-up link", func(t *testing.T) {
		fake := &fakePollSession{
			guildScheduledEventCreateEvent: &discordgo.GuildScheduledEvent{ID: "event-42"},
		}
		interaction := newGuildInteraction()

		if err := NativePoll(fake, interaction); err != nil {
			t.Fatalf("NativePoll() error = %v", err)
		}

		if len(fake.interactionRespondCalls) != 1 {
			t.Fatalf("InteractionRespond called %d times, want 1", len(fake.interactionRespondCalls))
		}
		if fake.interactionResponseCalls != 1 {
			t.Fatalf("InteractionResponse called %d times, want 1", fake.interactionResponseCalls)
		}
		if len(fake.guildScheduledEventCreateCalls) != 1 {
			t.Fatalf("GuildScheduledEventCreate called %d times, want 1", len(fake.guildScheduledEventCreateCalls))
		}
		call := fake.guildScheduledEventCreateCalls[0]
		if call.guildID != "guild-1" {
			t.Errorf("GuildScheduledEventCreate guildID = %q, want %q", call.guildID, "guild-1")
		}
		wantMessageURL := buildMessageURL("guild-1", "channel-1", "message-1")
		if call.params.EntityMetadata.Location != wantMessageURL {
			t.Errorf("EntityMetadata.Location = %q, want %q", call.params.EntityMetadata.Location, wantMessageURL)
		}

		if len(fake.followupMessageCreateCalls) != 1 {
			t.Fatalf("FollowupMessageCreate called %d times, want 1", len(fake.followupMessageCreateCalls))
		}
		wantEventURL := buildEventURL("guild-1", "event-42")
		if fake.followupMessageCreateCalls[0].Content != wantEventURL {
			t.Errorf("FollowupMessageCreate Content = %q, want %q", fake.followupMessageCreateCalls[0].Content, wantEventURL)
		}
	})

	t.Run("InteractionRespond error is returned and stops further calls", func(t *testing.T) {
		wantErr := errors.New("interaction respond failed")
		fake := &fakePollSession{interactionRespondErr: wantErr}
		interaction := newGuildInteraction()

		err := NativePoll(fake, interaction)
		if err != wantErr {
			t.Fatalf("NativePoll() error = %v, want %v", err, wantErr)
		}
		if fake.interactionResponseCalls != 0 {
			t.Error("InteractionResponse was called, want it not to be")
		}
		if len(fake.guildScheduledEventCreateCalls) != 0 {
			t.Error("GuildScheduledEventCreate was called, want it not to be")
		}
	})

	t.Run("InteractionResponse error is returned and stops further calls", func(t *testing.T) {
		wantErr := errors.New("interaction response failed")
		fake := &fakePollSession{interactionResponseErr: wantErr}
		interaction := newGuildInteraction()

		err := NativePoll(fake, interaction)
		if err != wantErr {
			t.Fatalf("NativePoll() error = %v, want %v", err, wantErr)
		}
		if len(fake.guildScheduledEventCreateCalls) != 0 {
			t.Error("GuildScheduledEventCreate was called, want it not to be")
		}
	})

	t.Run("GuildScheduledEventCreate error is only logged: NativePoll still returns nil and skips the follow-up", func(t *testing.T) {
		fake := &fakePollSession{guildScheduledEventCreateErr: errors.New("event create failed")}
		interaction := newGuildInteraction()

		if err := NativePoll(fake, interaction); err != nil {
			t.Fatalf("NativePoll() error = %v, want nil", err)
		}
		if len(fake.followupMessageCreateCalls) != 0 {
			t.Error("FollowupMessageCreate was called, want it not to be")
		}
	})

	t.Run("FollowupMessageCreate error is only logged: NativePoll still returns nil", func(t *testing.T) {
		fake := &fakePollSession{followupMessageCreateErr: errors.New("follow-up failed")}
		interaction := newGuildInteraction()

		if err := NativePoll(fake, interaction); err != nil {
			t.Fatalf("NativePoll() error = %v, want nil", err)
		}
	})
}
