package poll

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

const testBotUserID = "bot-1"

func botPollMessage(id, title string) *discordgo.Message {
	return &discordgo.Message{
		ID:        id,
		ChannelID: testPollChannelID,
		Author:    &discordgo.User{ID: testBotUserID},
		Poll:      &discordgo.Poll{Question: discordgo.PollMedia{Text: title}},
	}
}

func newAutocompleteInteraction(query string) *discordgo.Interaction {
	return &discordgo.Interaction{
		Type:      discordgo.InteractionApplicationCommandAutocomplete,
		ChannelID: testPollChannelID,
		Locale:    discordgo.EnglishUS,
		Member:    &discordgo.Member{User: &discordgo.User{ID: testCreatorID}},
		Data: discordgo.ApplicationCommandInteractionData{
			Name:        "poll-end",
			CommandType: discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandInteractionDataOption{
				{Name: "message", Type: discordgo.ApplicationCommandOptionString, Value: query},
			},
		},
	}
}

func autocompleteChoices(t *testing.T, fake *fakePollSession) []*discordgo.ApplicationCommandOptionChoice {
	t.Helper()
	if fake.interactionRespondArg == nil {
		t.Fatalf("no autocomplete response was sent")
	}
	if got := fake.interactionRespondArg.Type; got != discordgo.InteractionApplicationCommandAutocompleteResult {
		t.Fatalf("response type = %v, want an autocomplete result", got)
	}
	if fake.interactionRespondArg.Data == nil {
		t.Fatalf("autocomplete response has no data")
	}
	return fake.interactionRespondArg.Data.Choices
}

func choiceNames(choices []*discordgo.ApplicationCommandOptionChoice) []string {
	names := make([]string, 0, len(choices))
	for _, choice := range choices {
		names = append(names, choice.Name)
	}
	return names
}

func TestCandidates_Filters(t *testing.T) {
	otherAuthor := botPollMessage("300000000000000001", "他人の投票")
	otherAuthor.Author = &discordgo.User{ID: "someone-else"}

	notAPoll := botPollMessage("300000000000000002", "")
	notAPoll.Poll = nil

	ended := botPollMessage("300000000000000003", "終了済み")
	ended.Poll.Results = &discordgo.PollResults{Finalized: true}

	missingAuthor := botPollMessage("300000000000000004", "投稿者不明")
	missingAuthor.Author = nil

	wanted := botPollMessage(testPollMessageID, "予定調整")

	fake := &fakePollSession{channelMessagesResult: []*discordgo.Message{otherAuthor, notAPoll, ended, missingAuthor, wanted}}
	cache := newCandidateCache(time.Minute)

	got, err := cache.candidates(fake, testPollChannelID, testBotUserID)
	if err != nil {
		t.Fatalf("candidates returned unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("candidates = %+v, want only this bot's unfinished poll", got)
	}
	if got[0].messageID != testPollMessageID || got[0].title != "予定調整" {
		t.Errorf("candidate = %+v, want message %q titled %q", got[0], testPollMessageID, "予定調整")
	}
	if fake.channelMessagesLimit != scanLimit {
		t.Errorf("limit = %d, want %d", fake.channelMessagesLimit, scanLimit)
	}
}

// A nil element in the slice discordgo returns must not take the bot down.
func TestCandidates_SkipsNilMessages(t *testing.T) {
	wanted := botPollMessage(testPollMessageID, "予定調整")
	fake := &fakePollSession{channelMessagesResult: []*discordgo.Message{nil, wanted}}
	cache := newCandidateCache(time.Minute)

	got, err := cache.candidates(fake, testPollChannelID, testBotUserID)
	if err != nil {
		t.Fatalf("candidates returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].messageID != testPollMessageID {
		t.Errorf("candidates = %+v, want only the non-nil poll", got)
	}
}

func TestCandidates_CacheHit(t *testing.T) {
	fake := &fakePollSession{channelMessagesResult: []*discordgo.Message{botPollMessage(testPollMessageID, "予定調整")}}
	cache := newCandidateCache(time.Minute)

	for range 3 {
		if _, err := cache.candidates(fake, testPollChannelID, testBotUserID); err != nil {
			t.Fatalf("candidates returned unexpected error: %v", err)
		}
	}

	if got := len(fake.calls); got != 1 {
		t.Errorf("ChannelMessages was called %d times, want 1 within the TTL", got)
	}
}

func TestCandidates_CacheExpiry(t *testing.T) {
	fake := &fakePollSession{channelMessagesResult: []*discordgo.Message{botPollMessage(testPollMessageID, "予定調整")}}
	cache := newCandidateCache(time.Nanosecond)

	if _, err := cache.candidates(fake, testPollChannelID, testBotUserID); err != nil {
		t.Fatalf("candidates returned unexpected error: %v", err)
	}
	time.Sleep(time.Millisecond)
	if _, err := cache.candidates(fake, testPollChannelID, testBotUserID); err != nil {
		t.Fatalf("candidates returned unexpected error: %v", err)
	}

	if got := len(fake.calls); got != 2 {
		t.Errorf("ChannelMessages was called %d times, want 2 once the TTL passed", got)
	}
}

func TestCandidates_CachesPerChannel(t *testing.T) {
	fake := &fakePollSession{channelMessagesResult: []*discordgo.Message{botPollMessage(testPollMessageID, "予定調整")}}
	cache := newCandidateCache(time.Minute)

	if _, err := cache.candidates(fake, testPollChannelID, testBotUserID); err != nil {
		t.Fatalf("candidates returned unexpected error: %v", err)
	}
	if _, err := cache.candidates(fake, "999999999999999999", testBotUserID); err != nil {
		t.Fatalf("candidates returned unexpected error: %v", err)
	}

	if got := len(fake.calls); got != 2 {
		t.Errorf("ChannelMessages was called %d times, want one per channel", got)
	}
}

func TestCandidates_EvictsStaleChannels(t *testing.T) {
	fake := &fakePollSession{channelMessagesResult: []*discordgo.Message{botPollMessage(testPollMessageID, "予定調整")}}
	cache := newCandidateCache(time.Nanosecond)

	if _, err := cache.candidates(fake, "999999999999999999", testBotUserID); err != nil {
		t.Fatalf("candidates returned unexpected error: %v", err)
	}
	time.Sleep(time.Millisecond)
	if _, err := cache.candidates(fake, testPollChannelID, testBotUserID); err != nil {
		t.Fatalf("candidates returned unexpected error: %v", err)
	}

	if got := cache.size(); got != 1 {
		t.Errorf("cache holds %d channels, want the long-idle one dropped", got)
	}
}

func TestCandidates_ChannelMessagesError(t *testing.T) {
	wantErr := errors.New("channel messages failed")
	fake := &fakePollSession{channelMessagesErr: wantErr}
	cache := newCandidateCache(time.Minute)

	if _, err := cache.candidates(fake, testPollChannelID, testBotUserID); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if _, err := cache.candidates(fake, testPollChannelID, testBotUserID); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v (a failure must not be cached)", err, wantErr)
	}
	if got := len(fake.calls); got != 2 {
		t.Errorf("ChannelMessages was called %d times, want a retry after the failure", got)
	}
}

func TestAutocomplete_FiltersByQuery(t *testing.T) {
	fake := &fakePollSession{channelMessagesResult: []*discordgo.Message{
		botPollMessage("300000000000000001", "Sprint Planning"),
		botPollMessage("300000000000000002", "忘年会の日程"),
		botPollMessage("300000000000000003", "sprint retro"),
	}}

	cache := newCandidateCache(time.Minute)

	if err := cache.autocomplete(fake, newAutocompleteInteraction("SPRINT"), testBotUserID); err != nil {
		t.Fatalf("autocomplete returned unexpected error: %v", err)
	}

	choices := autocompleteChoices(t, fake)
	if len(choices) != 2 {
		t.Fatalf("choices = %v, want the two case-insensitive matches", choiceNames(choices))
	}
	for _, choice := range choices {
		if !strings.Contains(strings.ToLower(choice.Name), "sprint") {
			t.Errorf("choice %q does not match the query", choice.Name)
		}
	}
	if choices[0].Value != "300000000000000001" {
		t.Errorf("value = %v, want the message ID", choices[0].Value)
	}
}

func TestAutocomplete_EmptyQueryReturnsAll(t *testing.T) {
	fake := &fakePollSession{channelMessagesResult: []*discordgo.Message{
		botPollMessage("300000000000000001", "Sprint Planning"),
		botPollMessage("300000000000000002", "忘年会の日程"),
	}}

	cache := newCandidateCache(time.Minute)

	if err := cache.autocomplete(fake, newAutocompleteInteraction("  "), testBotUserID); err != nil {
		t.Fatalf("autocomplete returned unexpected error: %v", err)
	}

	if got := len(autocompleteChoices(t, fake)); got != 2 {
		t.Errorf("choices = %d, want all candidates", got)
	}
}

func TestAutocomplete_LimitsToDiscordMaximum(t *testing.T) {
	messages := make([]*discordgo.Message, 0, 30)
	for i := range 30 {
		messages = append(messages, botPollMessage(fmt.Sprintf("3000000000000000%02d", i), fmt.Sprintf("投票 %d", i)))
	}
	fake := &fakePollSession{channelMessagesResult: messages}
	cache := newCandidateCache(time.Minute)

	if err := cache.autocomplete(fake, newAutocompleteInteraction(""), testBotUserID); err != nil {
		t.Fatalf("autocomplete returned unexpected error: %v", err)
	}

	if got := len(autocompleteChoices(t, fake)); got != choiceLimit {
		t.Errorf("choices = %d, want %d", got, choiceLimit)
	}
}

func TestAutocomplete_TruncatesLongTitle(t *testing.T) {
	title := strings.Repeat("あ", 150)
	fake := &fakePollSession{channelMessagesResult: []*discordgo.Message{botPollMessage(testPollMessageID, title)}}
	cache := newCandidateCache(time.Minute)

	if err := cache.autocomplete(fake, newAutocompleteInteraction(""), testBotUserID); err != nil {
		t.Fatalf("autocomplete returned unexpected error: %v", err)
	}

	choices := autocompleteChoices(t, fake)
	if len(choices) != 1 {
		t.Fatalf("choices = %d, want 1", len(choices))
	}
	if got := len([]rune(choices[0].Name)); got != nameLengthLimit {
		t.Errorf("name length = %d runes, want %d", got, nameLengthLimit)
	}
}

func TestAutocomplete_ChannelMessagesError(t *testing.T) {
	fake := &fakePollSession{channelMessagesErr: errors.New("channel messages failed")}
	cache := newCandidateCache(time.Minute)

	if err := cache.autocomplete(fake, newAutocompleteInteraction("sprint"), testBotUserID); err != nil {
		t.Fatalf("autocomplete returned unexpected error: %v", err)
	}

	if got := autocompleteChoices(t, fake); len(got) != 0 {
		t.Errorf("choices = %v, want none when the fetch fails", choiceNames(got))
	}
}

func TestAutocomplete_MissingOption(t *testing.T) {
	fake := &fakePollSession{channelMessagesResult: []*discordgo.Message{botPollMessage(testPollMessageID, "予定調整")}}
	interaction := newAutocompleteInteraction("")
	interaction.Data = discordgo.ApplicationCommandInteractionData{
		Name:        "poll-end",
		CommandType: discordgo.ChatApplicationCommand,
	}

	cache := newCandidateCache(time.Minute)

	if err := cache.autocomplete(fake, interaction, testBotUserID); err != nil {
		t.Fatalf("autocomplete returned unexpected error: %v", err)
	}

	if got := autocompleteChoices(t, fake); len(got) != 0 {
		t.Errorf("choices = %v, want none", choiceNames(got))
	}
}
