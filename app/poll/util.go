package poll

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type I18n struct {
	Weekdays     []string
	Absence      string
	DefaultTitle string
	VotingPeriod string
	PollMessage  string
}

func GetI18n(lang discordgo.Locale) I18n {
	return I18n{
		Weekdays:     getWeekdays(lang),
		Absence:      getAbsence(lang),
		DefaultTitle: getTitle(lang),
		VotingPeriod: getVotingPeriod(lang),
		PollMessage:  getPollMessage(lang),
	}
}

func GetTimeZone(lang discordgo.Locale) (*time.Location, error) {
	timezone := map[discordgo.Locale]string{
		discordgo.Japanese: "Asia/Tokyo",
	}
	tz, ok := timezone[lang]
	if !ok {
		return time.Local, nil
	}
	// Can fail without host tzdata (e.g. a minimal self-hosted container);
	// too environment-dependent to reproduce in tests.
	return time.LoadLocation(tz)
}

func getWeekdays(lang discordgo.Locale) []string {
	localeWeekdays := map[discordgo.Locale][]string{
		discordgo.EnglishUS: {"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
		discordgo.Japanese:  {"日", "月", "火", "水", "木", "金", "土"},
	}
	weekdays, ok := localeWeekdays[lang]
	if !ok {
		return localeWeekdays[discordgo.EnglishUS]
	}
	return weekdays
}

func getAbsence(lang discordgo.Locale) string {
	absence := map[discordgo.Locale]string{
		discordgo.EnglishUS: "Absence",
		discordgo.Japanese:  "欠席",
	}
	name, ok := absence[lang]
	if !ok {
		return absence[discordgo.EnglishUS]
	}
	return name
}

func getTitle(lang discordgo.Locale) string {
	title := map[discordgo.Locale]string{
		discordgo.EnglishUS: "Poll",
		discordgo.Japanese:  "投票",
	}
	name, ok := title[lang]
	if !ok {
		return title[discordgo.EnglishUS]
	}
	return name
}

func getVotingPeriod(lang discordgo.Locale) string {
	votingPeriod := map[discordgo.Locale]string{
		discordgo.EnglishUS: "(🗳️Voting)",
		discordgo.Japanese:  "(🗳️投票期間中)",
	}
	name, ok := votingPeriod[lang]
	if !ok {
		return votingPeriod[discordgo.EnglishUS]
	}
	return name
}

func getPollMessage(lang discordgo.Locale) string {
	pollMessage := map[discordgo.Locale]string{
		discordgo.EnglishUS: "Poll message",
		discordgo.Japanese:  "投票メッセージ",
	}
	name, ok := pollMessage[lang]
	if !ok {
		return pollMessage[discordgo.EnglishUS]
	}
	return name
}

func FloatPtr(v float64) *float64 {
	return &v
}

func truncateRunes(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen])
}
