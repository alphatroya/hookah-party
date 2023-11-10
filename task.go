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
	draft     bool
	chatID    int64
	messageID int
	cancel    context.CancelFunc

	stageBegin    time.Time
	stageEnd      time.Time
	phaseDuration string

	isPaused       bool
	leftAfterPause time.Duration

	queue  *queue
	skipCh chan taskSkip
}

type taskSkip struct {
	resume bool
}

func NewTaskDraft(chatID int64, timeString string) *Task {
	t := new(Task)
	t.cancel = func() {}
	t.draft = true
	t.chatID = chatID
	if timeString == "" {
		t.phaseDuration = "160s"
	} else {
		t.phaseDuration = timeString
	}
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

func (t *Task) preview() string {
	preview := "Создаем покур:\n"
	if duration, err := time.ParseDuration(t.phaseDuration); err == nil {
		preview += "\tДлительность: " + durafmt.Parse(duration).String() + "\n"
	}
	if queue := t.queue; queue != nil && len(queue.users) != 0 {
		preview += "\tОчередь:\n"
		for i, user := range queue.users {
			preview += fmt.Sprintf("\t\t%d: %s", i+1, user)
			if queue.head == i {
				preview += " 🌬️"
			}
			preview += "\n"
		}
	}
	return preview
}

func (t *Task) addOrRemoveUserToQueue(user string) {
	if t.queue == nil {
		t.queue = new(queue)
	}
	for i, v := range t.queue.users {
		if v == user {
			t.queue.users = append(t.queue.users[:i], t.queue.users[i+1:]...)
			if t.queue.head == len(t.queue.users) {
				t.queue.head--
			}
			return
		}
	}
	t.queue.users = append(t.queue.users, user)
}

func (t *Task) Run(ctx context.Context, cancel context.CancelFunc, bot *tgbotapi.BotAPI) {
	t.cancel = cancel
	t.draft = false
	t.scheduleStage()

	bot.Send(tgbotapi.NewMessage(t.chatID, hookahStartedMsg+durafmt.Parse(t.stageEnd.Sub(t.stageBegin)).String()))
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
