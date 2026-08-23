package poll

import (
	"errors"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestGetDays(t *testing.T) {
	start := time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		numDays int
		want    []time.Time
	}{
		{
			name:    "single day returns just the start day",
			numDays: 1,
			want:    []time.Time{start},
		},
		{
			name:    "multiple days returns consecutive dates starting from start",
			numDays: 3,
			want: []time.Time{
				start,
				start.AddDate(0, 0, 1),
				start.AddDate(0, 0, 2),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDays(start, tt.numDays)
			if len(got) != len(tt.want) {
				t.Fatalf("getDays() returned %d days, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if !got[i].Equal(tt.want[i]) {
					t.Errorf("getDays()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetEmojis(t *testing.T) {
	got := getEmojis()
	// One numbered emoji per day of the week (1-7) plus the absence "❌"
	// option, since a poll offers at most 7 candidate days.
	if len(got) != 8 {
		t.Fatalf("getEmojis() returned %d emojis, want 8", len(got))
	}
	if got[7] != "❌" {
		t.Errorf("getEmojis()[7] = %q, want the absence emoji %q", got[7], "❌")
	}
}

func TestGetChoices(t *testing.T) {
	i18n := I18n{
		Weekdays: []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
		Absence:  "Absence",
	}
	// 2026-08-21 is a Friday.
	start := time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC)

	choices := getChoices(i18n, start, 3)

	if len(choices) != 4 {
		t.Fatalf("getChoices() returned %d choices, want 4 (3 days + absence)", len(choices))
	}
	want := []Choice{
		{Emoji: "1⃣", Name: "08/21 (Fri)"},
		{Emoji: "2⃣", Name: "08/22 (Sat)"},
		{Emoji: "3⃣", Name: "08/23 (Sun)"},
		{Emoji: "❌", Name: "Absence"},
	}
	for i, w := range want {
		if choices[i] != w {
			t.Errorf("getChoices()[%d] = %+v, want %+v", i, choices[i], w)
		}
	}
}

func TestResolveStartDate(t *testing.T) {
	// Fixed reference "now" so the year-rollover branch is exercised
	// deterministically instead of depending on the real wall clock.
	now := time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)

	stringOption := func(v string) *discordgo.ApplicationCommandInteractionDataOption {
		return &discordgo.ApplicationCommandInteractionDataOption{
			Type:  discordgo.ApplicationCommandOptionString,
			Value: v,
		}
	}

	tests := []struct {
		name string
		opt  *discordgo.ApplicationCommandInteractionDataOption
		want time.Time
	}{
		{
			name: "no option given returns today at midnight",
			opt:  nil,
			want: time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "date later this year than today is kept in the current year",
			opt:  stringOption("12/25"),
			want: time.Date(2026, time.December, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "date already passed this year rolls over to next year",
			opt:  stringOption("01/10"),
			want: time.Date(2027, time.January, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "today's own date is kept in the current year, not rolled over",
			opt:  stringOption("06/15"),
			want: time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "unparseable date falls back to today at midnight",
			opt:  stringOption("not-a-date"),
			want: time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveStartDate(now, time.UTC, tt.opt)
			if !got.Equal(tt.want) {
				t.Errorf("resolveStartDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newCommandInteraction(locale discordgo.Locale, options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.Interaction {
	return &discordgo.Interaction{
		Type:   discordgo.InteractionApplicationCommand,
		Locale: locale,
		Data: discordgo.ApplicationCommandInteractionData{
			Options: options,
		},
	}
}

func stringOpt(name, value string) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{
		Name:  name,
		Type:  discordgo.ApplicationCommandOptionString,
		Value: value,
	}
}

func intOpt(name string, value int) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{
		Name: name,
		Type: discordgo.ApplicationCommandOptionInteger,
		// Discord interaction payloads are JSON, so numeric option values
		// arrive as float64; IntValue() relies on that representation.
		Value: float64(value),
	}
}

func TestParsePollOptions(t *testing.T) {
	i18n := I18n{DefaultTitle: "Poll"}
	// A fixed reference "now", used everywhere below instead of the real
	// wall clock, so results are fully deterministic regardless of when or
	// how fast the test runs.
	fixedNow := time.Date(2026, time.June, 15, 9, 0, 0, 0, time.UTC)

	t.Run("title falls back to the default when not given", func(t *testing.T) {
		interaction := newCommandInteraction(discordgo.EnglishUS, nil)
		opts, err := parsePollOptions(interaction, i18n, fixedNow)
		if err != nil {
			t.Fatalf("parsePollOptions() error = %v", err)
		}
		if opts.Title != i18n.DefaultTitle {
			t.Errorf("Title = %q, want default %q", opts.Title, i18n.DefaultTitle)
		}
	})

	t.Run("title is used as-is when given", func(t *testing.T) {
		interaction := newCommandInteraction(discordgo.EnglishUS, []*discordgo.ApplicationCommandInteractionDataOption{
			stringOpt("title", "My Poll"),
		})
		opts, err := parsePollOptions(interaction, i18n, fixedNow)
		if err != nil {
			t.Fatalf("parsePollOptions() error = %v", err)
		}
		if opts.Title != "My Poll" {
			t.Errorf("Title = %q, want %q", opts.Title, "My Poll")
		}
	})

	t.Run("days defaults to 7 when not given", func(t *testing.T) {
		interaction := newCommandInteraction(discordgo.EnglishUS, nil)
		opts, err := parsePollOptions(interaction, i18n, fixedNow)
		if err != nil {
			t.Fatalf("parsePollOptions() error = %v", err)
		}
		if opts.NumDays != 7 {
			t.Errorf("NumDays = %d, want 7", opts.NumDays)
		}
	})

	daysClampTests := []struct {
		name  string
		given int
		want  int
	}{
		{"below minimum is clamped up to 2", 1, 2},
		{"above maximum is clamped down to 7", 9, 7},
		{"within range is kept as-is", 4, 4},
	}
	for _, tt := range daysClampTests {
		t.Run(tt.name, func(t *testing.T) {
			interaction := newCommandInteraction(discordgo.EnglishUS, []*discordgo.ApplicationCommandInteractionDataOption{
				intOpt("days", tt.given),
			})
			opts, err := parsePollOptions(interaction, i18n, fixedNow)
			if err != nil {
				t.Fatalf("parsePollOptions() error = %v", err)
			}
			if opts.NumDays != tt.want {
				t.Errorf("NumDays = %d, want %d", opts.NumDays, tt.want)
			}
		})
	}

	t.Run("start defaults to today at local midnight when no start-date given", func(t *testing.T) {
		timezone, err := GetTimeZone(discordgo.EnglishUS)
		if err != nil {
			t.Fatalf("GetTimeZone() error = %v", err)
		}
		localNow := fixedNow.In(timezone)
		wantStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, timezone)

		interaction := newCommandInteraction(discordgo.EnglishUS, nil)
		opts, err := parsePollOptions(interaction, i18n, fixedNow)
		if err != nil {
			t.Fatalf("parsePollOptions() error = %v", err)
		}
		if !opts.Start.Equal(wantStart) {
			t.Errorf("Start = %v, want %v", opts.Start, wantStart)
		}
	})

	t.Run("an unmapped locale falls back to time.Local instead of erroring", func(t *testing.T) {
		// GetTimeZone only special-cases discordgo.Japanese and otherwise
		// falls back to time.Local without error, so no discordgo.Locale
		// value currently reaches parsePollOptions' `if err != nil` branch;
		// this test only documents that fact, it does not exercise error
		// propagation.
		interaction := newCommandInteraction(discordgo.German, nil)
		_, err := parsePollOptions(interaction, i18n, fixedNow)
		if err != nil {
			t.Fatalf("parsePollOptions() error = %v, want nil", err)
		}
	})
}

func TestBuildMessageURL(t *testing.T) {
	got := buildMessageURL("guild1", "channel1", "message1")
	want := "https://discord.com/channels/guild1/channel1/message1"
	if got != want {
		t.Errorf("buildMessageURL() = %q, want %q", got, want)
	}
}

func TestBuildEventURL(t *testing.T) {
	got := buildEventURL("guild1", "event1")
	want := "https://discord.com/events/guild1/event1"
	if got != want {
		t.Errorf("buildEventURL() = %q, want %q", got, want)
	}
}

func TestEventStartTime(t *testing.T) {
	start := time.Date(2026, time.August, 21, 9, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		numDays int
		want    time.Time
	}{
		{
			name:    "single day event starts at midnight of that day",
			numDays: 1,
			want:    time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "multi-day event starts at midnight of the final candidate day",
			numDays: 3,
			want:    time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eventStartTime(start, tt.numDays)
			if !got.Equal(tt.want) {
				t.Errorf("eventStartTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveEventStartTime(t *testing.T) {
	start := time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC)

	t.Run("future midnight is kept as-is", func(t *testing.T) {
		now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
		got := resolveEventStartTime(start, 3, now)
		want := eventStartTime(start, 3) // 2026-08-23 midnight, still in the future
		if !got.Equal(want) {
			t.Errorf("resolveEventStartTime() = %v, want %v", got, want)
		}
	})

	t.Run("already-past midnight is bumped to now+1 minute", func(t *testing.T) {
		// eventStartTime(start, 1) is 2026-08-21 00:00, which has already
		// passed relative to "now" below.
		now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
		got := resolveEventStartTime(start, 1, now)
		want := now.Add(1 * time.Minute)
		if !got.Equal(want) {
			t.Errorf("resolveEventStartTime() = %v, want %v", got, want)
		}
	})
}

func TestClampPollDurationToEvent(t *testing.T) {
	eventStart := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		durationHours int
		now           time.Time
		want          int
	}{
		{
			name:          "duration within the remaining time is kept as-is",
			durationHours: 24,
			now:           time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC), // 120h remaining
			want:          24,
		},
		{
			name:          "duration longer than the remaining time is clamped down",
			durationHours: 240,
			now:           time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC), // 120h remaining
			want:          120,
		},
		{
			name:          "remaining time below the minimum floors to minPollDurationHours",
			durationHours: 24,
			now:           time.Date(2026, time.August, 24, 23, 30, 0, 0, time.UTC), // 0.5h remaining
			want:          minPollDurationHours,
		},
		{
			name:          "event start already passed floors to minPollDurationHours",
			durationHours: 24,
			now:           time.Date(2026, time.August, 26, 0, 0, 0, 0, time.UTC), // negative remaining
			want:          minPollDurationHours,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampPollDurationToEvent(tt.durationHours, eventStart, tt.now)
			if got != tt.want {
				t.Errorf("clampPollDurationToEvent() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildGuildScheduledEventParams(t *testing.T) {
	i18n := I18n{VotingPeriod: "(V)", PollMessage: "Msg"}
	start := time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC)
	numDays := 3
	messageURL := "https://discord.com/channels/guild-1/channel-1/message-1"
	eventStart := time.Date(2026, time.August, 23, 9, 0, 0, 0, time.UTC)

	params := buildGuildScheduledEventParams(i18n, start, numDays, "Test", messageURL, eventStart)

	if params.Name != "(V)Test" {
		t.Errorf("Name = %q, want %q", params.Name, "(V)Test")
	}
	wantDescription := "Msg: " + messageURL
	if params.Description != wantDescription {
		t.Errorf("Description = %q, want %q", params.Description, wantDescription)
	}
	if params.ScheduledStartTime == nil || !params.ScheduledStartTime.Equal(eventStart) {
		t.Errorf("ScheduledStartTime = %v, want %v", params.ScheduledStartTime, eventStart)
	}
	// The final candidate day is Aug 21 + 2 days = Aug 23; the event should
	// run through the end of that day.
	wantEnd := time.Date(2026, time.August, 23, 23, 59, 59, 0, time.UTC)
	if params.ScheduledEndTime == nil || !params.ScheduledEndTime.Equal(wantEnd) {
		t.Errorf("ScheduledEndTime = %v, want %v", params.ScheduledEndTime, wantEnd)
	}
	if params.PrivacyLevel != discordgo.GuildScheduledEventPrivacyLevelGuildOnly {
		t.Errorf("PrivacyLevel = %v, want %v", params.PrivacyLevel, discordgo.GuildScheduledEventPrivacyLevelGuildOnly)
	}
	if params.EntityType != discordgo.GuildScheduledEventEntityTypeExternal {
		t.Errorf("EntityType = %v, want %v", params.EntityType, discordgo.GuildScheduledEventEntityTypeExternal)
	}
	if params.EntityMetadata == nil || params.EntityMetadata.Location != messageURL {
		t.Errorf("EntityMetadata.Location = %v, want %q", params.EntityMetadata, messageURL)
	}

	t.Run("truncates a long title to Discord's event name limit", func(t *testing.T) {
		longTitle := ""
		for i := 0; i < discordEventNameMaxLength+10; i++ {
			longTitle += "a"
		}
		params := buildGuildScheduledEventParams(i18n, start, numDays, longTitle, messageURL, eventStart)
		if len([]rune(params.Name)) != discordEventNameMaxLength {
			t.Errorf("Name length = %d, want %d", len([]rune(params.Name)), discordEventNameMaxLength)
		}
	})
}

func TestCreateScheduledEvent(t *testing.T) {
	i18n := I18n{VotingPeriod: "(V)", PollMessage: "Msg"}
	start := time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC)
	eventStart := time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC)
	messageURL := "https://discord.com/channels/guild-1/channel-1/message-1"

	t.Run("delegates to GuildScheduledEventCreate with the built params and returns its result", func(t *testing.T) {
		fake := &fakePollSession{guildScheduledEventCreateEvent: &discordgo.GuildScheduledEvent{ID: "evt-42"}}

		got, err := createScheduledEvent(fake, "guild-1", i18n, start, 3, "Test", messageURL, eventStart)
		if err != nil {
			t.Fatalf("createScheduledEvent() error = %v", err)
		}
		if got.ID != "evt-42" {
			t.Errorf("returned event ID = %q, want %q", got.ID, "evt-42")
		}
		if len(fake.guildScheduledEventCreateCalls) != 1 {
			t.Fatalf("GuildScheduledEventCreate called %d times, want 1", len(fake.guildScheduledEventCreateCalls))
		}
		call := fake.guildScheduledEventCreateCalls[0]
		if call.guildID != "guild-1" {
			t.Errorf("guildID = %q, want %q", call.guildID, "guild-1")
		}
		if call.params.Name != "(V)Test" {
			t.Errorf("params.Name = %q, want %q", call.params.Name, "(V)Test")
		}
	})

	t.Run("propagates the error from GuildScheduledEventCreate", func(t *testing.T) {
		wantErr := errors.New("event create failed")
		fake := &fakePollSession{guildScheduledEventCreateErr: wantErr}

		_, err := createScheduledEvent(fake, "guild-1", i18n, start, 3, "Test", messageURL, eventStart)
		if err != wantErr {
			t.Errorf("createScheduledEvent() error = %v, want %v", err, wantErr)
		}
	})
}
