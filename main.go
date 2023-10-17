package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	var apiToken string
	flag.StringVar(&apiToken, "bot-api-key", "", "Bot API key")
	flag.Parse()

	if apiToken == "" {
		flag.Usage()
		os.Exit(1)
	}

	bot, err := tgbotapi.NewBotAPI(apiToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			if !update.Message.IsCommand() {
				continue
			}

			chatID := update.Message.Chat.ID
			switch update.Message.Command() {
			case "new":
				time := update.Message.CommandArguments()
				createNewTask(chatID, time, bot)
			case "cancel":
				tasks.cancel(chatID)
			case "pause":
				tasks.pause(chatID)
				bot.Send(tgbotapi.NewMessage(chatID, hookahPausedMsg))
			case "resume":
				tasks.resume(chatID)
				bot.Send(tgbotapi.NewMessage(chatID, hookahResumedMsg))
			case "skip":
				tasks.skip(chatID)
			case "help", "start":
				printHelp(chatID, bot)
			case "setparty":
				party := update.Message.CommandArguments()
				queue, err := newQueue(party)
				if err != nil {
					fmt.Printf("new party is wrong: %s\n", party)
				}
				err = tasks.setParty(chatID, queue)
				if err != nil {
					createNewTask(chatID, "", bot)
					_ = tasks.setParty(chatID, queue)
				}
				bot.Send(tgbotapi.NewMessage(chatID, queue.print()))
			}
		}
	}
}

func printHelp(chatID int64, bot *tgbotapi.BotAPI) {
	const help = `
- /new - запускает новую сессию кальяна и принимает параметр продолжительности цикла, например, 2м30с - каждый участник курит кальян по 2 минуты 30 секунд.
    - Если значение не указано, то используется значение по умолчанию - 2 минуты 40 секунд.

- /setparty - устанавливает очередь пользователей, например, "/setparty 1 2 3" устанавливает очередь из пользователей @1 @2 и @3.
    - При каждом срабатывании таймера бот обращается к текущему и следующему пользователю в очереди.
    - Если кальян не запущен, то он запускается с продолжительностью по умолчанию.

- /pause - приостанавливает сессию кальяна.

- /resume - возобновляет приостановленную сессию кальяна, и очередь текущего пользователя начинается сначала.

- /skip - сбрасывает текущего пользователя, и очередь переходит к следующему пользователю в очереди.
`
	bot.Send(tgbotapi.NewMessage(chatID, help))
}

func createNewTask(chatID int64, time string, bot *tgbotapi.BotAPI) {
	ctx, cancel := context.WithCancel(context.Background())
	t := NewTask(chatID, cancel, time)
	tasks.Place(chatID, t)
	t.Run(ctx, bot)
}
