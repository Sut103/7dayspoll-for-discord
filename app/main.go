package main

import (
	"7dayspoll/bot"
	"log"
	"os"
)

func main() {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	b := bot.NewBot(token)
	if err := b.Run(); err != nil {
		log.Fatalln(err)
	}
}
