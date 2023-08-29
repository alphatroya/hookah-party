package main

import (
	"context"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/hako/durafmt"
)

type TaskStage int

const (
	TaskStagePhase TaskStage = iota
)

func (t TaskStage) Message() string {
	return hookahNextMsg
}

type Task struct {
	taskStage      TaskStage
	chatID         int64
	cancel         context.CancelFunc
	nextStageDelay time.Duration
	isPaused       bool
	skipCh         chan struct {
		silent bool
	}
}

func NewTask(chatID int64, cancel context.CancelFunc, timeString string) *Task {
	t := new(Task)
	t.chatID = chatID
	t.cancel = cancel
	delay, err := time.ParseDuration(timeString)
	if err != nil {
		delay = 90 * time.Second
	}
	t.nextStageDelay = delay
	t.taskStage = TaskStagePhase
	t.skipCh = make(chan struct{ silent bool })
	return t
}

func (t *Task) Run(ctx context.Context, bot *tgbotapi.BotAPI) {
	bot.Send(tgbotapi.NewMessage(t.chatID, hookahStartedMsg+durafmt.Parse(t.nextStageDelay).String()))
	go func() {
		for {
			select {
			case s := <-t.skipCh:
				if !s.silent {
					bot.Send(tgbotapi.NewMessage(t.chatID, hookahStageSkippedMsg))
				}
			case <-time.After(t.nextStageDelay):
				if t.isPaused {
					continue
				}
				message := t.taskStage.Message()
				bot.Send(tgbotapi.NewMessage(t.chatID, message))
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (t *Task) pause() {
	t.isPaused = true
}

func (t *Task) resume() {
	t.isPaused = false
	t.skipCh <- struct {
		silent bool
	}{
		silent: true,
	}
}

func (t *Task) skip() {
	t.skipCh <- struct {
		silent bool
	}{}
}
