package main

import (
	"flag"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// TODO: add configuration flag --bot-api-key
	// TODO: configure logging
	// TODO: create single mode bot
	// TODO: support pause
	var apiToken string
	flag.StringVar(&apiToken, "bot-api-key", "", "Bot API key")
	flag.Parse()

	if apiToken == "" {
		// TODO: throw panic
		return
	}

	bot, err := tgbotapi.NewBotAPI(apiToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID

			go func() {
				time.Sleep(5 * time.Second)
				bot.Send(msg)
			}()
		}
	}
}
