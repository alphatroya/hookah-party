package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

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
				createNewTask(chatID, bot)
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
				tasks.setParty(chatID, queue)
				bot.Send(tgbotapi.NewMessage(chatID, hookahQueueMsg))
			}
		} else if update.CallbackQuery != nil {
			chatID := update.CallbackQuery.Message.Chat.ID
			handleCallback(bot, update.CallbackQuery, chatID)
		}
	}
}

const (
	queueCallbackPrefix    = "queue-"
	durationCallbackPrefix = "duration-"
)

func handleCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, chatID int64) {
	callbackData := query.Data

	switch callbackData {
	case "run":
		tasks.run(chatID, bot)
	case "cancel":
		tasks.cancel(chatID)
	default:
		if task, ok := tasks.get(chatID); ok {
			if strings.HasPrefix(callbackData, queueCallbackPrefix) {
				user := strings.TrimPrefix(callbackData, queueCallbackPrefix)
				task.addOrRemoveUserToQueue(user)
			} else if strings.HasPrefix(callbackData, durationCallbackPrefix) {
				duration := strings.TrimPrefix(callbackData, durationCallbackPrefix)
				task.phaseDuration = duration
			}
		}
	}

	if task, ok := tasks.get(chatID); ok {
		editMessage(bot, chatID, task.messageID, task.preview())
	}
}

func editMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int, newText string) {
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, newText)
	editMsg.ReplyMarkup = numericKeyboard(chatID)
	bot.Send(editMsg)
}

func printHelp(chatID int64, bot *tgbotapi.BotAPI) {
	// TODO: rewrite help
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

func createNewTask(chatID int64, bot *tgbotapi.BotAPI) {
	task := NewTaskDraft(chatID)
	tasks.Place(chatID, task)

	msg := tgbotapi.NewMessage(chatID, task.preview())
	msg.ReplyMarkup = numericKeyboard(chatID)
	if message, err := bot.Send(msg); err == nil {
		task.messageID = message.MessageID
	}
}

func numericKeyboard(chatID int64) *tgbotapi.InlineKeyboardMarkup {
	numericKeyboard := tgbotapi.NewInlineKeyboardMarkup()
	if queue := tasks.getParty(chatID); queue != nil {
		usersButton := make([]tgbotapi.InlineKeyboardButton, 0, len(queue.users))
		for _, user := range queue.users {
			usersButton = append(usersButton, tgbotapi.NewInlineKeyboardButtonData(user, queueCallbackPrefix+user))
		}
		numericKeyboard.InlineKeyboard = append(numericKeyboard.InlineKeyboard, usersButton)
	}

	if t, ok := tasks.get(chatID); ok {
		switch t.state {
		case stateDraft:
			numericKeyboard.InlineKeyboard = append(numericKeyboard.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData(durationStandard, durationCallbackPrefix+durationStandard),
				tgbotapi.NewInlineKeyboardButtonData(durationExtended, durationCallbackPrefix+durationExtended),
				tgbotapi.NewInlineKeyboardButtonData(durationLarge, durationCallbackPrefix+durationLarge),
			})
			runRow := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Run", "run"),
			)
			numericKeyboard.InlineKeyboard = append(numericKeyboard.InlineKeyboard, runRow)
		case stateRun:
			numericKeyboard.InlineKeyboard = append(numericKeyboard.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel"),
			})
		case stateCancelled:
		}
	}
	if len(numericKeyboard.InlineKeyboard) == 0 {
		return nil
	}
	return &numericKeyboard
}
