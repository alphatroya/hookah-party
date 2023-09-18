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

const TaskStagePhase TaskStage = iota

func (t TaskStage) Message(prev, next *string) string {
	if prev != nil && next != nil && *prev != *next {
		return fmt.Sprintf(hookahNextUserMsg, *prev, *next)
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
	skipCh         chan taskSkip
}

type taskSkip struct {
	resume bool
}

func NewTask(chatID int64, cancel context.CancelFunc, timeString string) *Task {
	t := new(Task)
	t.chatID = chatID
	t.cancel = cancel
	delay, err := time.ParseDuration(timeString)
	if err != nil {
		delay = 160 * time.Second
	}
	t.nextStageDelay = delay
	t.taskStage = TaskStagePhase
	t.skipCh = make(chan taskSkip)
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
					prev, next := t.queue.next()
					message = t.taskStage.Message(&next, &prev)
					message += "\n"
					message += "\n"
					message += t.queue.print()
				} else {
					message = t.taskStage.Message(nil, nil)
				}
				bot.Send(tgbotapi.NewMessage(t.chatID, message))
			}

			select {
			case s := <-t.skipCh:
				if s.resume {
					continue
				}
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
	t.skipCh <- taskSkip{resume: true}
}

func (t *Task) skip() {
	t.skipCh <- taskSkip{}
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
	filterEmpty := make([]string, 0, len(components))
	for _, component := range components {
		if component != "" {
			filterEmpty = append(filterEmpty, component)
		}
	}
	q := new(queue)
	q.users = filterEmpty
	return q, nil
}

func (q *queue) next() (string, string) {
	prev := q.users[q.head]
	q.head++
	if q.head >= len(q.users) {
		q.head = 0
	}
	return prev, q.users[q.head]
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
