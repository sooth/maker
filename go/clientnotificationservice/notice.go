// Copyright (C) 2018-2019 Cranky Kernel
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package clientnotificationservice

import "sync"

type Level string

const LevelInfo = "info"
const LevelWarning = "warning"
const LevelError = "error"

type Notice struct {
	Level   Level                  `json:"level"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

func NewNotice(level Level, msg string) *Notice {
	return &Notice{
		Level:   level,
		Message: msg,
	}
}

func (n *Notice) WithData(data map[string]interface{}) *Notice {
	n.Data = data
	return n
}

type Service struct {
	lock        sync.RWMutex
	subscribers map[chan *Notice]bool
}

func New() *Service {
	return &Service{
		subscribers: make(map[chan *Notice]bool),
	}
}

func (s *Service) Subscribe() chan *Notice {
	s.lock.Lock()
	defer s.lock.Unlock()
	channel := make(chan *Notice)
	s.subscribers[channel] = true
	return channel
}

func (s *Service) Unsubscribe(channel chan *Notice) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, exists := s.subscribers[channel]; !exists {
		return
	}
	delete(s.subscribers, channel)
}

func (s *Service) Broadcast(notice *Notice) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for channel := range s.subscribers {
		channel <- notice
	}
}
