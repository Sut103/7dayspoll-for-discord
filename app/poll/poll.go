package poll

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Choice struct {
	Emoji string
	Name  string
}

func getDays(day time.Time, numDays int) []time.Time {
	days := make([]time.Time, numDays)
	for i := range days {
		days[i] = day.AddDate(0, 0, i)
	}
	return days
}

func getEmojis() []string {
	return []string{
		"1⃣",
		"2⃣",
		"3⃣",
		"4⃣",
		"5⃣",
		"6⃣",
		"7⃣",
		"❌",
	}
}

func getChoices(i18n I18n, startDate time.Time, numDays int) []Choice {
	days := getDays(startDate, numDays)
	emojis := getEmojis()

	choices := []Choice{}
	for i := 0; i < numDays; i++ {
		choices = append(choices, Choice{
			Emoji: emojis[i],
			Name:  fmt.Sprintf("%s (%s)", days[i].Format("01/02"), i18n.Weekdays[days[i].Weekday()]),
		})
	}
	absence := Choice{
		Emoji: emojis[7],
		Name:  i18n.Absence,
	}
	choices = append(choices, absence)
	return choices
}

type pollOptions struct {
	Title   string
	Start   time.Time
	NumDays int
	OptMap  map[string]*discordgo.ApplicationCommandInteractionDataOption
}

func parsePollOptions(interaction *discordgo.Interaction, i18n I18n, now time.Time) (*pollOptions, error) {
	// get timezone
	timezone, err := GetTimeZone(interaction.Locale)
	if err != nil {
		log.Println(http.StatusInternalServerError, "timezone error", err)
		return nil, err
	}
	// get options
	options := interaction.ApplicationCommandData().Options
	optMap := map[string]*discordgo.ApplicationCommandInteractionDataOption{}
	for _, opt := range options {
		optMap[opt.Name] = opt
	}
	title := ""
	if t, ok := optMap["title"]; ok {
		title = t.StringValue()
	}
	if title == "" {
		title = i18n.DefaultTitle
	}
	// Get number of days (default: 7)
	numDays := 7
	if d, ok := optMap["days"]; ok {
		numDays = int(d.IntValue())
		if numDays < 2 {
			numDays = 2
		} else if numDays > 7 {
			numDays = 7
		}
	}
	// judgement start date
	start := resolveStartDate(now.In(timezone), timezone, optMap["start-date"])
	return &pollOptions{
		Title:   title,
		Start:   start,
		NumDays: numDays,
		OptMap:  optMap,
	}, nil
}

// resolveStartDate returns the poll's start date: today at local midnight in
// timezone, or, when dateOpt is a valid "MM/DD" start-date option, that
// day resolved to this year (or next year, if that day-of-year has already
// passed today). dateOpt may be nil, and an unparseable value falls back to
// today at local midnight.
func resolveStartDate(now time.Time, timezone *time.Location, dateOpt *discordgo.ApplicationCommandInteractionDataOption) time.Time {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, timezone)
	if dateOpt == nil {
		return start
	}
	yearDate := fmt.Sprintf("%d/%s", now.Year(), dateOpt.StringValue())
	yd, err := time.ParseInLocation("2006/01/02", yearDate, timezone)
	if err != nil {
		return start
	}
	if start.After(yd) {
		yd = yd.AddDate(1, 0, 0)
	}
	return yd
}

func buildMessageURL(guildID, channelID, messageID string) string {
	return fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID, channelID, messageID)
}

func buildEventURL(guildID, eventID string) string {
	return fmt.Sprintf("https://discord.com/events/%s/%s", guildID, eventID)
}

const discordEventNameMaxLength = 100

// eventStartTime returns the local midnight of the final candidate day. This
// is used as the guild scheduled event's start time, since once the
// scheduled start time passes the event begins automatically and its start
// time can no longer be updated after the date is decided.
func eventStartTime(start time.Time, numDays int) time.Time {
	days := getDays(start, numDays)
	finalCandidateDay := days[len(days)-1]
	return time.Date(finalCandidateDay.Year(), finalCandidateDay.Month(), finalCandidateDay.Day(), 0, 0, 0, 0, start.Location())
}

// resolveEventStartTime returns the guild scheduled event's start time: the
// local midnight of the final candidate day, bumped to now+1 minute if that
// midnight has already passed (Discord requires the scheduled start time to
// be in the future).
func resolveEventStartTime(start time.Time, numDays int, now time.Time) time.Time {
	startTime := eventStartTime(start, numDays)
	if startTime.Before(now) {
		startTime = now.Add(1 * time.Minute)
	}
	return startTime
}

// minPollDurationHours is the lower bound this bot enforces for a poll's
// duration when clamping it to fit before a linked scheduled event.
const minPollDurationHours = 1

// clampPollDurationToEvent floors durationHours so the poll's expiry never
// exceeds eventStart, guaranteeing pollDeadline <= eventStart. Falls back to
// minPollDurationHours instead of a non-positive value when eventStart is
// imminent or already passed.
func clampPollDurationToEvent(durationHours int, eventStart, now time.Time) int {
	remainingHours := int(eventStart.Sub(now).Hours()) // truncates toward zero == floor
	if remainingHours < minPollDurationHours {
		remainingHours = minPollDurationHours
	}
	if durationHours > remainingHours {
		durationHours = remainingHours
	}
	return durationHours
}

// buildGuildScheduledEventParams builds the parameters for the guild
// scheduled event linked to a poll: it runs from eventStart through the end
// of the final candidate day, and links back to the poll message via
// messageURL.
func buildGuildScheduledEventParams(i18n I18n, start time.Time, numDays int, title string, messageURL string, eventStart time.Time) *discordgo.GuildScheduledEventParams {
	eventTitle := truncateRunes(i18n.VotingPeriod+title, discordEventNameMaxLength)

	finalCandidateDayMidnight := eventStartTime(start, numDays)
	endTime := time.Date(finalCandidateDayMidnight.Year(), finalCandidateDayMidnight.Month(), finalCandidateDayMidnight.Day(), 23, 59, 59, 0, start.Location())

	return &discordgo.GuildScheduledEventParams{
		Name:               eventTitle,
		Description:        fmt.Sprintf("%s: %s", i18n.PollMessage, messageURL),
		ScheduledStartTime: &eventStart,
		ScheduledEndTime:   &endTime,
		PrivacyLevel:       discordgo.GuildScheduledEventPrivacyLevelGuildOnly,
		EntityType:         discordgo.GuildScheduledEventEntityTypeExternal,
		EntityMetadata: &discordgo.GuildScheduledEventEntityMetadata{
			Location: messageURL,
		},
	}
}

func createScheduledEvent(session pollSession, guildID string, i18n I18n, start time.Time, numDays int, title string, messageURL string, eventStart time.Time) (*discordgo.GuildScheduledEvent, error) {
	eventParams := buildGuildScheduledEventParams(i18n, start, numDays, title, messageURL, eventStart)
	return session.GuildScheduledEventCreate(guildID, eventParams)
}
