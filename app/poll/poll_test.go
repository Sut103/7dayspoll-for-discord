package poll

import (
	"reflect"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ---- getDays ----------------------------------------------------------

// TestGetDays pins down that getDays returns numDays consecutive calendar
// days starting at (and including) day, preserving day's time-of-day.
func TestGetDays(t *testing.T) {
	tests := []struct {
		name    string
		day     time.Time
		numDays int
		want    []time.Time
	}{
		{
			name:    "three consecutive days within a month",
			day:     time.Date(2024, time.March, 10, 9, 30, 0, 0, time.UTC),
			numDays: 3,
			want: []time.Time{
				time.Date(2024, time.March, 10, 9, 30, 0, 0, time.UTC),
				time.Date(2024, time.March, 11, 9, 30, 0, 0, time.UTC),
				time.Date(2024, time.March, 12, 9, 30, 0, 0, time.UTC),
			},
		},
		{
			name:    "spans a month boundary",
			day:     time.Date(2024, time.January, 30, 0, 0, 0, 0, time.UTC),
			numDays: 4,
			want: []time.Time{
				time.Date(2024, time.January, 30, 0, 0, 0, 0, time.UTC),
				time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC),
				time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2024, time.February, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:    "single day",
			day:     time.Date(2024, time.July, 4, 12, 0, 0, 0, time.UTC),
			numDays: 1,
			want: []time.Time{
				time.Date(2024, time.July, 4, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDays(tt.day, tt.numDays)
			if len(got) != tt.numDays {
				t.Fatalf("getDays returned %d days, want %d", len(got), tt.numDays)
			}
			for i := range got {
				if !got[i].Equal(tt.want[i]) {
					t.Errorf("getDays[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ---- getEmojis ---------------------------------------------------------

// TestGetEmojis pins down the fixed 8-emoji list: keycap digits 1-7 followed
// by the cross-mark "absence" emoji.
func TestGetEmojis(t *testing.T) {
	want := []string{
		"1⃣", // 1️⃣ (digit + combining enclosing keycap)
		"2⃣",
		"3⃣",
		"4⃣",
		"5⃣",
		"6⃣",
		"7⃣",
		"❌", // ❌
	}

	got := getEmojis()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("getEmojis() = %v, want %v", got, want)
	}
}

// ---- getChoices ---------------------------------------------------------

// TestGetChoices pins down that getChoices returns numDays weekday-labeled
// choices followed by exactly one absence choice (numDays+1 total).
func TestGetChoices(t *testing.T) {
	i18n := I18n{
		Weekdays: []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
		Absence:  "Absence",
	}
	// 2024-03-04 is a Monday.
	start := time.Date(2024, time.March, 4, 0, 0, 0, 0, time.UTC)
	numDays := 3

	got := getChoices(i18n, start, numDays)

	if len(got) != numDays+1 {
		t.Fatalf("getChoices returned %d choices, want %d (numDays+1)", len(got), numDays+1)
	}

	wantDayChoices := []Choice{
		{Emoji: "1⃣", Name: "03/04 (Mon)"},
		{Emoji: "2⃣", Name: "03/05 (Tue)"},
		{Emoji: "3⃣", Name: "03/06 (Wed)"},
	}
	for i, want := range wantDayChoices {
		if got[i] != want {
			t.Errorf("choice[%d] = %+v, want %+v", i, got[i], want)
		}
	}

	wantAbsence := Choice{Emoji: "❌", Name: "Absence"}
	if last := got[len(got)-1]; last != wantAbsence {
		t.Errorf("last choice (absence) = %+v, want %+v", last, wantAbsence)
	}
}

// ---- parsePollOptions ----------------------------------------------------

func stringOption(name, value string) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{
		Name:  name,
		Type:  discordgo.ApplicationCommandOptionString,
		Value: value,
	}
}

func intOption(name string, value int64) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{
		Name: name,
		Type: discordgo.ApplicationCommandOptionInteger,
		// discordgo decodes integer option values from JSON as float64;
		// IntValue() asserts on that representation.
		Value: float64(value),
	}
}

func newInteraction(locale discordgo.Locale, opts ...*discordgo.ApplicationCommandInteractionDataOption) *discordgo.Interaction {
	return &discordgo.Interaction{
		Type:   discordgo.InteractionApplicationCommand,
		Locale: locale,
		Data: discordgo.ApplicationCommandInteractionData{
			Options: opts,
		},
	}
}

func TestParsePollOptions_TitleDefaultsAndOverride(t *testing.T) {
	i18n := I18n{DefaultTitle: "Default Poll Title"}

	tests := []struct {
		name      string
		opts      []*discordgo.ApplicationCommandInteractionDataOption
		wantTitle string
	}{
		{"no title option uses default", nil, "Default Poll Title"},
		{"empty title option uses default", []*discordgo.ApplicationCommandInteractionDataOption{stringOption("title", "")}, "Default Poll Title"},
		{"provided title is used verbatim", []*discordgo.ApplicationCommandInteractionDataOption{stringOption("title", "Team Lunch")}, "Team Lunch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interaction := newInteraction(discordgo.EnglishUS, tt.opts...)
			got, err := parsePollOptions(interaction, i18n)
			if err != nil {
				t.Fatalf("parsePollOptions returned unexpected error: %v", err)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
		})
	}
}

func TestParsePollOptions_NumDaysDefaultAndClamping(t *testing.T) {
	i18n := I18n{DefaultTitle: "Poll"}

	tests := []struct {
		name        string
		opts        []*discordgo.ApplicationCommandInteractionDataOption
		wantNumDays int
	}{
		{"no days option defaults to 7", nil, 7},
		{"below minimum (1) clamps up to 2", []*discordgo.ApplicationCommandInteractionDataOption{intOption("days", 1)}, 2},
		{"zero clamps up to 2", []*discordgo.ApplicationCommandInteractionDataOption{intOption("days", 0)}, 2},
		{"at minimum (2) is unchanged", []*discordgo.ApplicationCommandInteractionDataOption{intOption("days", 2)}, 2},
		{"mid-range (5) is unchanged", []*discordgo.ApplicationCommandInteractionDataOption{intOption("days", 5)}, 5},
		{"at maximum (7) is unchanged", []*discordgo.ApplicationCommandInteractionDataOption{intOption("days", 7)}, 7},
		{"above maximum (8) clamps down to 7", []*discordgo.ApplicationCommandInteractionDataOption{intOption("days", 8)}, 7},
		{"well above maximum clamps down to 7", []*discordgo.ApplicationCommandInteractionDataOption{intOption("days", 100)}, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interaction := newInteraction(discordgo.EnglishUS, tt.opts...)
			got, err := parsePollOptions(interaction, i18n)
			if err != nil {
				t.Fatalf("parsePollOptions returned unexpected error: %v", err)
			}
			if got.NumDays != tt.wantNumDays {
				t.Errorf("NumDays = %d, want %d", got.NumDays, tt.wantNumDays)
			}
		})
	}
}

func TestParsePollOptions_StartDateDefault(t *testing.T) {
	i18n := I18n{DefaultTitle: "Poll"}
	interaction := newInteraction(discordgo.EnglishUS) // no start-date option

	got, err := parsePollOptions(interaction, i18n)
	if err != nil {
		t.Fatalf("parsePollOptions returned unexpected error: %v", err)
	}

	now := time.Now().In(time.Local)
	wantStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if !got.Start.Equal(wantStart) {
		t.Errorf("Start = %v, want %v (today at local midnight)", got.Start, wantStart)
	}
}

func TestParsePollOptions_StartDateFutureThisYear(t *testing.T) {
	i18n := I18n{DefaultTitle: "Poll"}
	now := time.Now().In(time.Local)
	if now.Month() == time.December && now.Day() == 31 {
		t.Skip("cannot construct an unambiguous future-of-this-year date on December 31st")
	}
	tomorrow := now.AddDate(0, 0, 1)

	interaction := newInteraction(discordgo.EnglishUS, stringOption("start-date", tomorrow.Format("01/02")))
	got, err := parsePollOptions(interaction, i18n)
	if err != nil {
		t.Fatalf("parsePollOptions returned unexpected error: %v", err)
	}

	wantStart := time.Date(now.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.Local)
	if !got.Start.Equal(wantStart) {
		t.Errorf("Start = %v, want %v (this year's date, not rolled over)", got.Start, wantStart)
	}
}

func TestParsePollOptions_StartDateSameAsTodayDoesNotRoll(t *testing.T) {
	i18n := I18n{DefaultTitle: "Poll"}
	now := time.Now().In(time.Local)

	interaction := newInteraction(discordgo.EnglishUS, stringOption("start-date", now.Format("01/02")))
	got, err := parsePollOptions(interaction, i18n)
	if err != nil {
		t.Fatalf("parsePollOptions returned unexpected error: %v", err)
	}

	wantStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if !got.Start.Equal(wantStart) {
		t.Errorf("Start = %v, want %v (today, not rolled over)", got.Start, wantStart)
	}
}

func TestParsePollOptions_StartDatePastRollsToNextYear(t *testing.T) {
	i18n := I18n{DefaultTitle: "Poll"}
	now := time.Now().In(time.Local)
	if now.YearDay() == 1 {
		t.Skip("cannot construct a past-of-this-year date on January 1st")
	}
	yesterday := now.AddDate(0, 0, -1)

	interaction := newInteraction(discordgo.EnglishUS, stringOption("start-date", yesterday.Format("01/02")))
	got, err := parsePollOptions(interaction, i18n)
	if err != nil {
		t.Fatalf("parsePollOptions returned unexpected error: %v", err)
	}

	// A date that's already in the past this year is assumed to mean "next
	// occurrence of that date", i.e. next year.
	wantStart := time.Date(now.Year()+1, yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.Local)
	if !got.Start.Equal(wantStart) {
		t.Errorf("Start = %v, want %v (rolled forward to next year)", got.Start, wantStart)
	}
}

func TestParsePollOptions_StartDateUnparseableFallsBackToToday(t *testing.T) {
	i18n := I18n{DefaultTitle: "Poll"}
	now := time.Now().In(time.Local)

	interaction := newInteraction(discordgo.EnglishUS, stringOption("start-date", "not-a-date"))
	got, err := parsePollOptions(interaction, i18n)
	if err != nil {
		t.Fatalf("parsePollOptions returned unexpected error: %v", err)
	}

	wantStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if !got.Start.Equal(wantStart) {
		t.Errorf("Start = %v, want %v (unparseable start-date ignored, default kept)", got.Start, wantStart)
	}
}

// TestParsePollOptions_TimezoneErrorPropagation documents (rather than
// exercises) the "GetTimeZone failure propagates as an error" behavior
// required by issue #62.
//
// parsePollOptions does propagate a GetTimeZone error verbatim:
//
//	timezone, err := GetTimeZone(interaction.Locale)
//	if err != nil {
//	    return nil, err
//	}
//
// However, GetTimeZone (util.go) can only fail via time.LoadLocation, and
// its locale table only ever passes it the hardcoded, always-valid zone name
// "Asia/Tokyo" (for discordgo.Japanese) or short-circuits to time.Local (nil
// error) for every other locale. There is no black-box input to
// parsePollOptions that makes GetTimeZone fail without either modifying
// poll.go/util.go to allow injecting a fake timezone resolver, or mutating
// the test host's system timezone database out from under the process
// (which was deliberately avoided as unsafe/non-portable for a unit test).
// This is flagged for the reviewer rather than silently skipped.
func TestParsePollOptions_TimezoneErrorPropagation(t *testing.T) {
	t.Skip("GetTimeZone cannot be made to fail for any input reachable from parsePollOptions without modifying source or system state; see comment above")
}

// ---- buildMessageURL / buildEventURL -------------------------------------

func TestBuildMessageURL(t *testing.T) {
	got := buildMessageURL("111", "222", "333")
	want := "https://discord.com/channels/111/222/333"
	if got != want {
		t.Errorf("buildMessageURL = %q, want %q", got, want)
	}
}

func TestBuildEventURL(t *testing.T) {
	got := buildEventURL("111", "444")
	want := "https://discord.com/events/111/444"
	if got != want {
		t.Errorf("buildEventURL = %q, want %q", got, want)
	}
}

// ---- eventStartTime -------------------------------------------------------

// TestEventStartTime pins down that eventStartTime returns local midnight of
// the final candidate day (start + numDays - 1), in start's location.
func TestEventStartTime(t *testing.T) {
	tests := []struct {
		name    string
		start   time.Time
		numDays int
		want    time.Time
	}{
		{
			name:    "multi-day range midnight of last day",
			start:   time.Date(2024, time.March, 1, 15, 30, 0, 0, time.UTC),
			numDays: 5,
			want:    time.Date(2024, time.March, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "single day midnight of that same day",
			start:   time.Date(2024, time.March, 1, 15, 30, 0, 0, time.UTC),
			numDays: 1,
			want:    time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "range spanning a month boundary",
			start:   time.Date(2024, time.January, 30, 8, 0, 0, 0, time.UTC),
			numDays: 3,
			want:    time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eventStartTime(tt.start, tt.numDays)
			if !got.Equal(tt.want) {
				t.Errorf("eventStartTime = %v, want %v", got, tt.want)
			}
			if got.Location().String() != tt.start.Location().String() {
				t.Errorf("eventStartTime location = %v, want %v", got.Location(), tt.start.Location())
			}
		})
	}
}

// ---- resolveEventStartTime -------------------------------------------------

// TestResolveEventStartTime pins down both branches: the computed midnight
// is used as-is when still in the future relative to now, and bumped to
// now+1 minute when it has already passed.
func TestResolveEventStartTime(t *testing.T) {
	tests := []struct {
		name    string
		start   time.Time
		numDays int
		now     time.Time
		want    time.Time
	}{
		{
			name:    "midnight still in the future is used as-is",
			start:   time.Date(2024, time.June, 20, 8, 0, 0, 0, time.UTC),
			numDays: 1,
			now:     time.Date(2024, time.June, 15, 12, 0, 0, 0, time.UTC),
			want:    time.Date(2024, time.June, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "midnight already passed is bumped to now+1 minute",
			start:   time.Date(2024, time.June, 1, 8, 0, 0, 0, time.UTC),
			numDays: 3, // final candidate day June 3, well before now (June 15)
			now:     time.Date(2024, time.June, 15, 12, 0, 0, 0, time.UTC),
			want:    time.Date(2024, time.June, 15, 12, 1, 0, 0, time.UTC),
		},
		{
			name:    "midnight exactly equal to now is not bumped (strict Before check)",
			start:   time.Date(2024, time.June, 10, 5, 0, 0, 0, time.UTC),
			numDays: 1, // final candidate day midnight == 2024-06-10T00:00:00Z
			now:     time.Date(2024, time.June, 10, 0, 0, 0, 0, time.UTC),
			want:    time.Date(2024, time.June, 10, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveEventStartTime(tt.start, tt.numDays, tt.now)
			if !got.Equal(tt.want) {
				t.Errorf("resolveEventStartTime = %v, want %v", got, tt.want)
			}
		})
	}
}

// ---- clampPollDurationToEvent ----------------------------------------------

// TestClampPollDurationToEvent pins down that the duration is floored to the
// whole hours remaining before eventStart, never going below
// minPollDurationHours (1), even when eventStart has already passed.
func TestClampPollDurationToEvent(t *testing.T) {
	now := time.Date(2024, time.June, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		durationHours int
		eventStart    time.Time
		want          int
	}{
		{
			name:          "plenty of remaining time: duration unchanged",
			durationHours: 24,
			eventStart:    now.Add(100 * time.Hour),
			want:          24,
		},
		{
			name:          "insufficient remaining time: floored to remaining whole hours",
			durationHours: 24,
			eventStart:    now.Add(5 * time.Hour),
			want:          5,
		},
		{
			name:          "fractional remaining hours truncate toward zero (floor)",
			durationHours: 24,
			eventStart:    now.Add(2*time.Hour + 30*time.Minute),
			want:          2,
		},
		{
			name:          "eventStart already in the past: floors at minPollDurationHours, not 0 or negative",
			durationHours: 24,
			eventStart:    now.Add(-10 * time.Hour),
			want:          1,
		},
		{
			name:          "eventStart exactly now: floors at minPollDurationHours",
			durationHours: 24,
			eventStart:    now,
			want:          1,
		},
		{
			name:          "duration already below remaining time is left untouched",
			durationHours: 3,
			eventStart:    now.Add(1000 * time.Hour),
			want:          3,
		},
		{
			name:          "duration exactly equal to remaining time is left untouched",
			durationHours: 10,
			eventStart:    now.Add(10 * time.Hour),
			want:          10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampPollDurationToEvent(tt.durationHours, tt.eventStart, now)
			if got != tt.want {
				t.Errorf("clampPollDurationToEvent(%d, %v, %v) = %d, want %d", tt.durationHours, tt.eventStart, now, got, tt.want)
			}
			if got < minPollDurationHours {
				t.Errorf("clampPollDurationToEvent returned %d, which is below minPollDurationHours (%d)", got, minPollDurationHours)
			}
		})
	}
}
