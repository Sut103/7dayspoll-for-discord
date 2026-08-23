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

type commandKind int

const (
	commandUnknown commandKind = iota
	commandNativePoll
	commandClassicPoll
)

// resolveCommandKind maps an application command's name to the kind of poll
// command it is, or commandUnknown for a name this bot doesn't handle.
func resolveCommandKind(name string) commandKind {
	switch name {
	case "poll":
		return commandNativePoll
	case "poll-classic":
		return commandClassicPoll
	default:
		return commandUnknown
	}
}

func botHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	var err error
	switch resolveCommandKind(i.ApplicationCommandData().Name) {
	case commandNativePoll:
		err = poll.NativePoll(s, i.Interaction)
	case commandClassicPoll:
		err = poll.ClassicPoll(s, i.Interaction)
	default:
		return
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
