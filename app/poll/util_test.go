package poll

import (
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestGetTimeZone(t *testing.T) {
	tests := []struct {
		name string
		lang discordgo.Locale
		want *time.Location
	}{
		{"japanese maps to Asia/Tokyo", discordgo.Japanese, mustLoadLocation(t, "Asia/Tokyo")},
		{"english (US) falls back to local", discordgo.EnglishUS, time.Local},
		{"unrecognized locale falls back to local", discordgo.Locale("xx-XX"), time.Local},
		{"empty locale falls back to local", discordgo.Locale(""), time.Local},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTimeZone(tt.lang)
			if err != nil {
				t.Fatalf("GetTimeZone(%q) returned unexpected error: %v", tt.lang, err)
			}
			if got.String() != tt.want.String() {
				t.Errorf("GetTimeZone(%q) = %v, want %v", tt.lang, got, tt.want)
			}
		})
	}
}

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("failed to load location %q for test setup: %v", name, err)
	}
	return loc
}

func TestGetI18n(t *testing.T) {
	enUS := I18n{
		Weekdays:     []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
		Absence:      "Absence",
		DefaultTitle: "Poll",
		VotingPeriod: "(🗳️Voting)",
		PollMessage:  "Poll message",
	}
	japanese := I18n{
		Weekdays:     []string{"日", "月", "火", "水", "木", "金", "土"},
		Absence:      "欠席",
		DefaultTitle: "投票",
		VotingPeriod: "(🗳️投票期間中)",
		PollMessage:  "投票メッセージ",
	}

	tests := []struct {
		name string
		lang discordgo.Locale
		want I18n
	}{
		{"EnglishUS", discordgo.EnglishUS, enUS},
		{"Japanese", discordgo.Japanese, japanese},
		{"unrecognized locale falls back to EnglishUS set", discordgo.Locale("fr"), enUS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetI18n(tt.lang)
			if !equalI18n(got, tt.want) {
				t.Errorf("GetI18n(%q) = %+v, want %+v", tt.lang, got, tt.want)
			}
		})
	}
}

func equalI18n(a, b I18n) bool {
	if a.Absence != b.Absence || a.DefaultTitle != b.DefaultTitle || a.VotingPeriod != b.VotingPeriod || a.PollMessage != b.PollMessage {
		return false
	}
	if len(a.Weekdays) != len(b.Weekdays) {
		return false
	}
	for i := range a.Weekdays {
		if a.Weekdays[i] != b.Weekdays[i] {
			return false
		}
	}
	return true
}

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"ascii under maxLen is unchanged", "hello", 10, "hello"},
		{"ascii exactly at maxLen is unchanged", "hello", 5, "hello"},
		{"ascii over maxLen is truncated", "hello world", 5, "hello"},
		{"empty string is unchanged", "", 5, ""},
		{"maxLen zero on non-empty string truncates to empty", "hello", 0, ""},
		{
			name:   "japanese string exactly at maxLen (rune count) is unchanged",
			s:      "こんにちは", // 5 runes, 15 bytes
			maxLen: 5,
			want:   "こんにちは",
		},
		{
			name:   "japanese string over maxLen truncates by rune count without corrupting characters",
			s:      "こんにちは世界", // 7 runes, 21 bytes
			maxLen: 5,
			want:   "こんにちは",
		},
		{
			name:   "mixed ascii+japanese truncates by rune count",
			s:      "Hi こんにちは", // 8 runes: H,i,' ',こ,ん,に,ち,は
			maxLen: 3,
			want:   "Hi ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateRunes(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateRunes(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
			if gotRunes := len([]rune(got)); gotRunes > tt.maxLen && tt.maxLen >= 0 {
				t.Errorf("truncateRunes(%q, %d) returned %d runes, want <= %d", tt.s, tt.maxLen, gotRunes, tt.maxLen)
			}
			if !utf8ValidRunesRoundTrip(got) {
				t.Errorf("truncateRunes(%q, %d) = %q is not valid UTF-8 (character corrupted)", tt.s, tt.maxLen, got)
			}
		})
	}
}

// A byte-level cut mid-character would surface as U+FFFD here.
func utf8ValidRunesRoundTrip(s string) bool {
	return !strings.ContainsRune(s, '�')
}
