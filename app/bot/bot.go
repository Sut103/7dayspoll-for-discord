package bot

import (
	"7dayspoll/manage"
	"7dayspoll/poll"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	token string
}

func NewBot(token string) *Bot {
	return &Bot{
		token,
	}
}

const (
	commandKindNative  = "native"
	commandKindClassic = "classic"
)

// resolveCommandKind maps a Discord application command name to the kind of
// poll it should trigger. It is a pure function with no dependency on
// discordgo types, so it can be unit-tested in isolation from botHandler's
// dispatch, which requires a live *discordgo.Session/*discordgo.InteractionCreate.
func resolveCommandKind(name string) (kind string, ok bool) {
	switch name {
	case "poll":
		return commandKindNative, true
	case "poll-classic":
		return commandKindClassic, true
	default:
		return "", false
	}
}

func botHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	kind, ok := resolveCommandKind(i.ApplicationCommandData().Name)
	if !ok {
		return
	}
	var err error
	switch kind {
	case commandKindNative:
		err = poll.NativePoll(s, i.Interaction)
	case commandKindClassic:
		err = poll.ClassicPoll(s, i.Interaction)
	}
	if err != nil {
		log.Println(err)
	}
}

func messageReactionAddEventHandler(s *discordgo.Session, event *discordgo.MessageReactionAdd) {
	ctx := poll.NewAggregationContext(event.ChannelID, event.MessageID)
	err := poll.AggregatePoll(ctx, s, event.MessageReaction)
	if err != nil {
		log.Println(err)
		return
	}
}
func messageReactionRemoveEventHandler(s *discordgo.Session, event *discordgo.MessageReactionRemove) {
	ctx := poll.NewAggregationContext(event.ChannelID, event.MessageID)
	err := poll.AggregatePoll(ctx, s, event.MessageReaction)
	if err != nil {
		log.Println(err)
		return
	}
}

func (b *Bot) Run() error {
	s, err := discordgo.New(fmt.Sprintf("%s %s", "Bot", b.token))
	if err != nil {
		return err
	}

	s.AddHandler(botHandler)
	s.AddHandler(messageReactionAddEventHandler)
	s.AddHandler(messageReactionRemoveEventHandler)
	err = s.Open()
	if err != nil {
		return err
	}
	defer s.Close()

	manage.Register(s, s.State.User.ID)

	log.Println("=====start=====")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(
		signalChan,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	<-signalChan
	return nil
}
