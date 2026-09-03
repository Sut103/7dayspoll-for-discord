package poll

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestNativePoll_DM_HappyPath(t *testing.T) {
	fake := &fakePollSession{}
	interaction := newInteraction(discordgo.EnglishUS)
	interaction.GuildID = ""

	err := NativePoll(fake, interaction)
	if err != nil {
		t.Fatalf("NativePoll returned unexpected error: %v", err)
	}

	wantCalls := []string{"InteractionRespond"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v (guild-only calls must not happen in a DM)", fake.calls, wantCalls)
	}

	gotDuration := fake.interactionRespondArg.Data.Poll.Duration
	wantDuration := defaultDurationDays * 24
	if gotDuration != wantDuration {
		t.Errorf("Duration = %d, want %d (unclamped default)", gotDuration, wantDuration)
	}
}

func TestNativePoll_InteractionRespondError(t *testing.T) {
	wantErr := errors.New("interaction respond failed")
	fake := &fakePollSession{interactionRespondErr: wantErr}
	interaction := newInteraction(discordgo.EnglishUS)
	interaction.GuildID = "guild-1"

	err := NativePoll(fake, interaction)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}

	wantCalls := []string{"InteractionRespond"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
}

func TestNativePoll_Guild_HappyPath(t *testing.T) {
	fake := &fakePollSession{
		interactionResponseResult:       &discordgo.Message{ID: "msg-1"},
		guildScheduledEventCreateResult: &discordgo.GuildScheduledEvent{ID: "event-1"},
	}
	// days=2 puts the event within the default 72h duration, forcing an actual clamp.
	interaction := newInteraction(discordgo.EnglishUS, intOption("days", 2))
	interaction.GuildID = "guild-1"
	interaction.ChannelID = "chan-1"

	now := time.Now()
	err := NativePoll(fake, interaction)
	if err != nil {
		t.Fatalf("NativePoll returned unexpected error: %v", err)
	}

	wantCalls := []string{"InteractionRespond", "InteractionResponse", "GuildScheduledEventCreate", "FollowupMessageCreate"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}

	i18n := GetI18n(discordgo.EnglishUS)
	opts, err := parsePollOptions(interaction, i18n)
	if err != nil {
		t.Fatalf("parsePollOptions returned unexpected error: %v", err)
	}
	wantEventStart := resolveEventStartTime(opts.Start, opts.NumDays, now)
	wantDuration := clampPollDurationToEvent(defaultDurationDays*24, wantEventStart, now)

	gotDuration := fake.interactionRespondArg.Data.Poll.Duration
	if gotDuration != wantDuration {
		t.Errorf("Duration = %d, want %d (clamped to the linked event's start)", gotDuration, wantDuration)
	}
	if gotDuration >= defaultDurationDays*24 {
		t.Errorf("Duration = %d, want it clamped below the unclamped default %d", gotDuration, defaultDurationDays*24)
	}

	if fake.guildScheduledEventCreateGuildID != "guild-1" {
		t.Errorf("GuildScheduledEventCreate guildID = %q, want %q", fake.guildScheduledEventCreateGuildID, "guild-1")
	}
	if fake.guildScheduledEventCreateParams == nil || fake.guildScheduledEventCreateParams.ScheduledStartTime == nil ||
		!fake.guildScheduledEventCreateParams.ScheduledStartTime.Equal(wantEventStart) {
		t.Errorf("GuildScheduledEventCreate ScheduledStartTime = %v, want %v", fake.guildScheduledEventCreateParams, wantEventStart)
	}

	wantContent := buildEventURL("guild-1", "event-1")
	if fake.followupMessageCreateData == nil || fake.followupMessageCreateData.Content != wantContent {
		t.Errorf("follow-up content = %+v, want Content=%q", fake.followupMessageCreateData, wantContent)
	}
}

func TestNativePoll_InteractionResponseError(t *testing.T) {
	wantErr := errors.New("interaction response failed")
	fake := &fakePollSession{interactionResponseErr: wantErr}
	interaction := newInteraction(discordgo.EnglishUS)
	interaction.GuildID = "guild-1"

	err := NativePoll(fake, interaction)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}

	wantCalls := []string{"InteractionRespond", "InteractionResponse"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v (GuildScheduledEventCreate/FollowupMessageCreate must not run)", fake.calls, wantCalls)
	}
}

func TestNativePoll_GuildScheduledEventCreateError_IsSwallowed(t *testing.T) {
	fake := &fakePollSession{
		interactionResponseResult:    &discordgo.Message{ID: "msg-1"},
		guildScheduledEventCreateErr: errors.New("event create failed"),
	}
	interaction := newInteraction(discordgo.EnglishUS)
	interaction.GuildID = "guild-1"

	err := NativePoll(fake, interaction)
	if err != nil {
		t.Fatalf("NativePoll returned %v, want nil (event-creation errors are logged and swallowed)", err)
	}

	wantCalls := []string{"InteractionRespond", "InteractionResponse", "GuildScheduledEventCreate"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v (FollowupMessageCreate must not run)", fake.calls, wantCalls)
	}
}

func TestNativePoll_FollowupMessageCreateError_IsSwallowed(t *testing.T) {
	fake := &fakePollSession{
		interactionResponseResult:       &discordgo.Message{ID: "msg-1"},
		guildScheduledEventCreateResult: &discordgo.GuildScheduledEvent{ID: "event-1"},
		followupMessageCreateErr:        errors.New("followup failed"),
	}
	interaction := newInteraction(discordgo.EnglishUS)
	interaction.GuildID = "guild-1"

	err := NativePoll(fake, interaction)
	if err != nil {
		t.Fatalf("NativePoll returned %v, want nil (follow-up errors are logged and swallowed)", err)
	}

	wantCalls := []string{"InteractionRespond", "InteractionResponse", "GuildScheduledEventCreate", "FollowupMessageCreate"}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
}

func TestCreateScheduledEvent_Params(t *testing.T) {
	fake := &fakePollSession{
		guildScheduledEventCreateResult: &discordgo.GuildScheduledEvent{ID: "event-1"},
	}
	i18n := I18n{VotingPeriod: "(Voting) ", PollMessage: "Poll message"}
	start := time.Date(2024, time.March, 1, 9, 0, 0, 0, time.UTC)
	numDays := 3
	title := strings.Repeat("a", 150) // long enough that Name truncation is exercised
	messageURL := "https://discord.com/channels/guild-1/chan-1/msg-1"
	eventStart := time.Date(2024, time.March, 3, 0, 0, 0, 0, time.UTC)

	_, err := createScheduledEvent(fake, "guild-1", i18n, start, numDays, title, messageURL, eventStart)
	if err != nil {
		t.Fatalf("createScheduledEvent returned unexpected error: %v", err)
	}

	if fake.guildScheduledEventCreateGuildID != "guild-1" {
		t.Errorf("guildID = %q, want %q", fake.guildScheduledEventCreateGuildID, "guild-1")
	}

	got := fake.guildScheduledEventCreateParams
	if got == nil {
		t.Fatalf("GuildScheduledEventCreate was not called with params")
	}

	wantName := truncateRunes(i18n.VotingPeriod+title, discordEventNameMaxLength)
	if got.Name != wantName {
		t.Errorf("Name = %q, want %q", got.Name, wantName)
	}
	if n := len([]rune(got.Name)); n > discordEventNameMaxLength {
		t.Errorf("Name has %d runes, want <= %d", n, discordEventNameMaxLength)
	}

	wantDescription := fmt.Sprintf("%s: %s", i18n.PollMessage, messageURL)
	if got.Description != wantDescription {
		t.Errorf("Description = %q, want %q", got.Description, wantDescription)
	}

	if got.ScheduledStartTime == nil || !got.ScheduledStartTime.Equal(eventStart) {
		t.Errorf("ScheduledStartTime = %v, want %v", got.ScheduledStartTime, eventStart)
	}

	wantEnd := time.Date(2024, time.March, 3, 23, 59, 59, 0, start.Location())
	if got.ScheduledEndTime == nil || !got.ScheduledEndTime.Equal(wantEnd) {
		t.Errorf("ScheduledEndTime = %v, want %v", got.ScheduledEndTime, wantEnd)
	}

	if got.PrivacyLevel != discordgo.GuildScheduledEventPrivacyLevelGuildOnly {
		t.Errorf("PrivacyLevel = %v, want %v", got.PrivacyLevel, discordgo.GuildScheduledEventPrivacyLevelGuildOnly)
	}
	if got.EntityType != discordgo.GuildScheduledEventEntityTypeExternal {
		t.Errorf("EntityType = %v, want %v", got.EntityType, discordgo.GuildScheduledEventEntityTypeExternal)
	}
	if got.EntityMetadata == nil || got.EntityMetadata.Location != messageURL {
		t.Errorf("EntityMetadata.Location = %+v, want %q", got.EntityMetadata, messageURL)
	}
}
