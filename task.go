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
	taskStage TaskStage
	chatID    int64
	cancel    context.CancelFunc

	stageBegin    time.Time
	stageEnd      time.Time
	phaseDuration string
	lastMessageID *int

	isPaused       bool
	leftAfterPause time.Duration

	queue  *queue
	skipCh chan taskSkip
}

type taskSkip struct {
	resume bool
}

func NewTask(chatID int64, cancel context.CancelFunc, timeString string) *Task {
	t := new(Task)
	t.chatID = chatID
	t.cancel = cancel
	t.phaseDuration = timeString
	t.scheduleStage()
	t.taskStage = TaskStagePhase
	t.skipCh = make(chan taskSkip)
	return t
}

func (t *Task) scheduleStage() {
	t.stageBegin = time.Now()
	delay, err := time.ParseDuration(t.phaseDuration)
	if err != nil {
		delay = 160 * time.Second
	}
	t.stageEnd = t.stageBegin.Add(delay)
}

func (t *Task) Run(ctx context.Context, bot *tgbotapi.BotAPI) {
	bot.Send(tgbotapi.NewMessage(t.chatID, hookahStartedMsg+durafmt.Parse(t.stageEnd.Sub(t.stageBegin)).String()))
	go func() {
		for {
			f := func() {
				if t.isPaused {
					return
				}
				if lastMessageID := t.lastMessageID; lastMessageID != nil {
					bot.Send(tgbotapi.NewDeleteMessage(t.chatID, *lastMessageID))
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
				if messageInfo, err := bot.Send(tgbotapi.NewMessage(t.chatID, message)); err == nil {
					t.lastMessageID = &messageInfo.MessageID
				}
				t.scheduleStage()
			}

			select {
			case s := <-t.skipCh:
				if s.resume {
					continue
				}
				f()
			case <-time.After(t.stageEnd.Sub(t.stageBegin)):
				f()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (t *Task) pause() {
	t.leftAfterPause = time.Until(t.stageEnd)
	t.isPaused = true
}

func (t *Task) resume() {
	t.isPaused = false
	t.stageBegin = time.Now()
	t.stageEnd = t.stageBegin.Add(t.leftAfterPause)
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
