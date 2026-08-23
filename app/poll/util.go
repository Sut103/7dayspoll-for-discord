package poll

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type I18n struct {
	Weekdays              []string
	Absence               string
	DefaultTitle          string
	VotingPeriod          string
	PollMessage           string
	PollNotFound          string
	PollAlreadyEnded      string
	PollEndNoPermission   string
	PollEndFailed         string
	PollEndSuccess        string
	PollEndInvalidMessage string
}

func GetI18n(lang discordgo.Locale) I18n {
	return I18n{
		Weekdays:              getWeekdays(lang),
		Absence:               getAbsence(lang),
		DefaultTitle:          getTitle(lang),
		VotingPeriod:          getVotingPeriod(lang),
		PollMessage:           getPollMessage(lang),
		PollNotFound:          getPollNotFound(lang),
		PollAlreadyEnded:      getPollAlreadyEnded(lang),
		PollEndNoPermission:   getPollEndNoPermission(lang),
		PollEndFailed:         getPollEndFailed(lang),
		PollEndSuccess:        getPollEndSuccess(lang),
		PollEndInvalidMessage: getPollEndInvalidMessage(lang),
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

func getPollNotFound(lang discordgo.Locale) string {
	pollNotFound := map[discordgo.Locale]string{
		discordgo.EnglishUS: "This message doesn't have a poll.",
		discordgo.Japanese:  "このメッセージには投票がありません。",
	}
	name, ok := pollNotFound[lang]
	if !ok {
		return pollNotFound[discordgo.EnglishUS]
	}
	return name
}

func getPollAlreadyEnded(lang discordgo.Locale) string {
	pollAlreadyEnded := map[discordgo.Locale]string{
		discordgo.EnglishUS: "This poll has already ended.",
		discordgo.Japanese:  "この投票はすでに終了しています。",
	}
	name, ok := pollAlreadyEnded[lang]
	if !ok {
		return pollAlreadyEnded[discordgo.EnglishUS]
	}
	return name
}

func getPollEndNoPermission(lang discordgo.Locale) string {
	pollEndNoPermission := map[discordgo.Locale]string{
		discordgo.EnglishUS: "Only the poll's creator or a member with Manage Messages can end it.",
		discordgo.Japanese:  "投票を終了できるのは投稿者か「メッセージの管理」権限を持つメンバーのみです。",
	}
	name, ok := pollEndNoPermission[lang]
	if !ok {
		return pollEndNoPermission[discordgo.EnglishUS]
	}
	return name
}

func getPollEndFailed(lang discordgo.Locale) string {
	pollEndFailed := map[discordgo.Locale]string{
		discordgo.EnglishUS: "Failed to end the poll. Please try again.",
		discordgo.Japanese:  "投票の終了に失敗しました。もう一度お試しください。",
	}
	name, ok := pollEndFailed[lang]
	if !ok {
		return pollEndFailed[discordgo.EnglishUS]
	}
	return name
}

func getPollEndSuccess(lang discordgo.Locale) string {
	pollEndSuccess := map[discordgo.Locale]string{
		discordgo.EnglishUS: "The poll has been ended.",
		discordgo.Japanese:  "投票を終了しました。",
	}
	name, ok := pollEndSuccess[lang]
	if !ok {
		return pollEndSuccess[discordgo.EnglishUS]
	}
	return name
}

func getPollEndInvalidMessage(lang discordgo.Locale) string {
	pollEndInvalidMessage := map[discordgo.Locale]string{
		discordgo.EnglishUS: "That doesn't look like a message in this channel. Use a message link or ID from this channel.",
		discordgo.Japanese:  "このチャンネル内のメッセージとして認識できませんでした。このチャンネルのメッセージリンクかIDを指定してください。",
	}
	name, ok := pollEndInvalidMessage[lang]
	if !ok {
		return pollEndInvalidMessage[discordgo.EnglishUS]
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
