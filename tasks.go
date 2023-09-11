package main

import (
	"errors"
	"sync"
)

var tasks = &storage{Mutex: new(sync.Mutex), x: make(map[int64]*Task)}

type storage struct {
	*sync.Mutex
	x map[int64]*Task
}

func (s *storage) Place(chat int64, t *Task) {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.x[chat]; ok {
		prev.cancel()
	}
	s.x[chat] = t
}

func (s *storage) cancel(chat int64) {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.x[chat]; ok {
		prev.cancel()
	}
}

func (s *storage) pause(chat int64) {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.x[chat]; ok {
		prev.pause()
	}
}

func (s *storage) resume(chat int64) {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.x[chat]; ok {
		prev.resume()
	}
}

func (s *storage) setParty(chat int64, party *queue) error {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.x[chat]; ok {
		prev.queue = party
		return nil
	}
	return errors.New("кальян и не запущен")
}

func (s *storage) skip(chat int64) {
	s.Lock()
	defer s.Unlock()
	if prev, ok := s.x[chat]; ok {
		prev.skip()
	}
}
