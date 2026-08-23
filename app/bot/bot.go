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

func botHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	var err error
	switch i.ApplicationCommandData().Name {
	case "poll":
		err = poll.NativePoll(s, i.Interaction)
	case "poll-classic":
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

	manage.Register(s)

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
