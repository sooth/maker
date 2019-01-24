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

package binanceex

import (
	"encoding/json"
	"fmt"
	"gitlab.com/crankykernel/cryptotrader/binance"
	"gitlab.com/crankykernel/maker/go/log"
	"strings"
	"sync"
	"time"
)

type TickerStreamChannel chan binance.StreamTicker24

type TickerStreamManager struct {
	mutex         sync.RWMutex
	subscriptions map[TickerStreamChannel]bool
	streams       map[string]*binance.StreamClient
	streamCount   map[string]int
}

func NewTickerStreamManager() *TickerStreamManager {
	return &TickerStreamManager{
		subscriptions: make(map[TickerStreamChannel]bool),
		streams:       make(map[string]*binance.StreamClient),
		streamCount:   make(map[string]int),
	}
}

func (m *TickerStreamManager) Subscribe() TickerStreamChannel {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	channel := make(TickerStreamChannel)
	m.subscriptions[channel] = true
	return channel
}

func (m *TickerStreamManager) Unsubscribe(channel TickerStreamChannel) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, exists := m.subscriptions[channel]; !exists {
		log.Errorf("Attempt to unsubscribe non existing channel")
	}
	m.subscriptions[channel] = false
	delete(m.subscriptions, channel)
}

func (m *TickerStreamManager) AddTicker(ticker string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, exists := m.streamCount[ticker]
	if exists {
		m.streamCount[ticker] += 1
		return
	}
	m.streamCount[ticker] = 1
	go m.runStream(ticker)
}

func (m *TickerStreamManager) RemoveTicker(ticker string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	count, exists := m.streamCount[ticker]
	if !exists {
		return
	}
	if count > 1 {
		m.streamCount[ticker] -= 1
	} else {
		delete(m.streamCount, ticker)
	}
}

func (m *TickerStreamManager) streamRefCount(name string) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	count, exists := m.streamCount[name]
	if exists {
		return count
	}
	return 0
}

func (m *TickerStreamManager) runStream(name string) {
Retry:
	if m.streamRefCount(name) == 0 {
		return
	}
	streamName := fmt.Sprintf("%s@ticker", strings.ToLower(name))
	stream, err := binance.OpenStream(streamName)
	if err != nil {
		log.WithError(err).
			WithField("stream", streamName).
			Errorf("Failed to open ticker stream")
		time.Sleep(1 * time.Second)
		goto Retry
	}
	for {
		_, payload, err := stream.Next()
		if err != nil {
			log.WithError(err).
				WithField("stream", streamName).
				Errorf("Failed to read ticker stream message")
			stream.Close()
			time.Sleep(1 * time.Second)
			goto Retry
		}

		// Check if we still have subscribers.
		count := m.streamRefCount(name)
		if count == 0 {
			log.WithFields(log.Fields{
				"tickerStream": name,
			}).Infof("Ticker reference count is zero, disconnected stream")
			stream.Close()
			return
		}

		var ticker binance.StreamTicker24
		if err := json.Unmarshal(payload, &ticker); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"name": name,
			}).Errorf("Failed to decode ticker stream message")
			continue
		}

		m.mutex.RLock()
		for channel := range m.subscriptions {
			channel <- ticker
		}
		m.mutex.RUnlock()
	}
}
