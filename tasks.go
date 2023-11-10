package main

import (
	"context"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var tasks = &storage{Mutex: new(sync.Mutex), tasks: make(map[int64]*Task), parties: make(map[int64]*queue)}

type storage struct {
	*sync.Mutex
	tasks   map[int64]*Task
	parties map[int64]*queue
}

func (s *storage) Place(chat int64, t *Task) {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.tasks[chat]; ok {
		prev.cancel()
	}
	s.tasks[chat] = t
}

func (s *storage) cancel(chatID int64) {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.tasks[chatID]; ok {
		prev.cancel()
		delete(s.tasks, chatID)
	}
}

func (s *storage) pause(chat int64) {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.tasks[chat]; ok {
		prev.pause()
	}
}

func (s *storage) get(chat int64) (*Task, bool) {
	s.Lock()
	defer s.Unlock()
	t, ok := s.tasks[chat]
	return t, ok
}

func (s *storage) run(chat int64, bot *tgbotapi.BotAPI) {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.tasks[chat]; ok {
		ctx, cancel := context.WithCancel(context.Background())
		prev.Run(ctx, cancel, bot)
	}
}

func (s *storage) resume(chat int64) {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.tasks[chat]; ok {
		prev.resume()
	}
}

func (s *storage) setParty(chat int64, party *queue) {
	s.Lock()
	defer s.Unlock()
	s.parties[chat] = party
}

func (s *storage) getParty(chat int64) *queue {
	s.Lock()
	defer s.Unlock()
	return s.parties[chat]
}

func (s *storage) skip(chat int64) {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.tasks[chat]; ok {
		prev.skip()
	}
}
