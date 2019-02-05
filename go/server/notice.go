// Copyright (C) 2018 Cranky Kernel
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

package server

import "sync"

type ClientNoticeLevel string

const ClientNoticeLevelWarning = "warning"

type ClientNotice struct {
	Level   ClientNoticeLevel `json:"level"`
	Message string            `json:"message"`
}

func NewClientNotice(level ClientNoticeLevel, msg string) ClientNotice {
	return ClientNotice{
		Level:   level,
		Message: msg,
	}
}

type ClientNoticeService struct {
	lock        sync.RWMutex
	subscribers map[chan ClientNotice]bool
}

func NewClientNoticeService() *ClientNoticeService {
	return &ClientNoticeService{
		subscribers: make(map[chan ClientNotice]bool),
	}
}

func (s *ClientNoticeService) Subscribe() chan ClientNotice {
	s.lock.Lock()
	defer s.lock.Unlock()
	channel := make(chan ClientNotice)
	s.subscribers[channel] = true
	return channel
}

func (s *ClientNoticeService) Unsubscribe(channel chan ClientNotice) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, exists := s.subscribers[channel]; !exists {
		return
	}
	delete(s.subscribers, channel)
}

func (s *ClientNoticeService) Broadcast(notice ClientNotice) {
	s.lock.RLock()
	for channel := range s.subscribers {
		channel <- notice
	}
	s.lock.RUnlock()
}
