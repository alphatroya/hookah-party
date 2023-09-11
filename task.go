package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/hako/durafmt"
)

type TaskStage int

const (
	TaskStagePhase TaskStage = iota
)

func (t TaskStage) Message(name *string) string {
	if name != nil {
		return fmt.Sprintf(hookahNextUserMsg, *name)
	}
	return hookahNextMsg
}

type Task struct {
	taskStage      TaskStage
	chatID         int64
	cancel         context.CancelFunc
	nextStageDelay time.Duration
	isPaused       bool
	queue          *queue
	skipCh         chan struct{}
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
	t.skipCh = make(chan struct{})
	return t
}

func (t *Task) Run(ctx context.Context, bot *tgbotapi.BotAPI) {
	bot.Send(tgbotapi.NewMessage(t.chatID, hookahStartedMsg+durafmt.Parse(t.nextStageDelay).String()))
	go func() {
		for {
			f := func() {
				if t.isPaused {
					return
				}
				var message string
				if t.queue != nil {
					next := "@" + t.queue.next()
					message = t.taskStage.Message(&next)
					message += "\n"
					message += "\n"
					message += t.queue.print()
				} else {
					message = t.taskStage.Message(nil)
				}
				bot.Send(tgbotapi.NewMessage(t.chatID, message))
			}

			select {
			case <-t.skipCh:
				f()
			case <-time.After(t.nextStageDelay):
				f()
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
	t.skipCh <- struct{}{}
}

func (t *Task) skip() {
	t.skipCh <- struct{}{}
}

type queue struct {
	users []string
	head  int
}

func newQueue(command string) (*queue, error) {
	components := strings.Split(command, " ")
	if len(components) == 0 {
		return nil, fmt.Errorf("queue is empty, command=%s", command)
	}
	q := new(queue)
	q.users = components
	return q, nil
}

func (q *queue) next() string {
	q.head++
	if q.head >= len(q.users) {
		q.head = 0
	}
	return q.users[q.head]
}

func (q *queue) print() string {
	if len(q.users) == 0 {
		return ""
	}
	var builder strings.Builder
	for i, user := range q.users {
		if i == q.head {
			builder.WriteString(user + " 🌬️" + "\n")
			continue
		}
		builder.WriteString(user + "\n")
	}
	return builder.String()
}
