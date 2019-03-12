// Copyright (C) 2019 Cranky Kernel
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

package healthservice

import "sync"

type State struct {
	BinanceUserSocketState string `json:"binanceUserSocketState"`
}

type Service struct {
	lock        sync.RWMutex
	subscribers map[chan *State]bool
	state       State
}

func New() *Service {
	return &Service{
		subscribers: make(map[chan *State]bool),
		state:       State{},
	}
}

func (s *Service) Update(cb func(state *State)) {
	s.lock.Lock()
	cb(&s.state)
	s.lock.Unlock()
	s.Broadcast()
}

func (s *Service) Subscribe() chan *State {
	s.lock.Lock()
	defer s.lock.Unlock()
	channel := make(chan *State, 1)
	channel <- &s.state
	s.subscribers[channel] = true
	return channel
}

func (s *Service) Unsubscribe(channel chan *State) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, exists := s.subscribers[channel]; !exists {
		return
	}
	delete(s.subscribers, channel)
}

func (s *Service) Broadcast() {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for channel := range s.subscribers {
		channel <- &s.state
	}
}
