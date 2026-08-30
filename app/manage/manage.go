package manage

import (
	"log"

	"7dayspoll/poll"

	"github.com/bwmarrin/discordgo"
)

func Register(session commandSession, appID string) error {
	log.Printf("Registering commands...\n")
	commands := []*discordgo.ApplicationCommand{
		poll.GetNativePollCommand(),
		poll.GetClassicPollCommand(),
	}
	for _, command := range commands {
		_, err := session.ApplicationCommandCreate(appID, "", command)
		if err != nil {
			return err
		}
	}
	log.Printf("Command registration completed successfully.\n")
	return nil
}

func Delete(session commandSession, appID string) error {
	log.Printf("Deleting registered commands...\n")
	commands, err := session.ApplicationCommands(appID, "")
	if err != nil {
		return err
	}

	if len(commands) < 1 {
		log.Println("Could not find commands")
		return nil
	}

	for _, command := range commands {
		err := session.ApplicationCommandDelete(appID, "", command.ID)
		if err != nil {
			return err
		}
	}
	log.Printf("Deletion completed successfully. Deleted %d commands\n", len(commands))
	return nil
}
