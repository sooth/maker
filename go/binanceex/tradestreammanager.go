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
	"github.com/crankykernel/binanceapi-go"
	"gitlab.com/crankykernel/maker/go/log"
	"strings"
	"sync"
	"time"
)

type TradeStreamChannel chan binanceapi.StreamAggTrade

type TradeStreamManager struct {
	lock          sync.RWMutex
	subscriptions map[TradeStreamChannel]string
	streams       map[string]*binanceapi.Stream
	streamCount   map[string]int
}

func NewTradeStreamManager() *TradeStreamManager {
	return &TradeStreamManager{
		subscriptions: make(map[TradeStreamChannel]string),
		streams:       make(map[string]*binanceapi.Stream),
		streamCount:   make(map[string]int),
	}
}

func (m *TradeStreamManager) Subscribe(name string) TradeStreamChannel {
	m.lock.Lock()
	defer m.lock.Unlock()
	channel := make(TradeStreamChannel, 3)
	m.subscriptions[channel] = name
	return channel
}

func (m *TradeStreamManager) Unsubscribe(channel TradeStreamChannel) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, exists := m.subscriptions[channel]; !exists {
		log.Errorf("Attempt to unsubscribe non existing channel")
	}
	delete(m.subscriptions, channel)
}

func (m *TradeStreamManager) AddSymbol(symbol string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	symbol = strings.ToLower(symbol)
	_, exists := m.streamCount[symbol]
	if exists {
		m.streamCount[symbol] += 1
		return
	}
	m.streamCount[symbol] = 1
	go m.runStream(symbol)
}

func (m *TradeStreamManager) RemoveSymbol(symbol string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	symbol = strings.ToLower(symbol)
	count, exists := m.streamCount[symbol]
	if !exists {
		return
	}
	if count > 1 {
		m.streamCount[symbol] -= 1
	} else {
		delete(m.streamCount, symbol)
	}
}

func (m *TradeStreamManager) streamRefCount(name string) int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	count, exists := m.streamCount[name]
	if exists {
		return count
	}
	return 0
}

func (m *TradeStreamManager) runStream(name string) {
Retry:
	if m.streamRefCount(name) == 0 {
		return
	}
	streamName := fmt.Sprintf("%s@aggTrade", strings.ToLower(name))
	stream, err := binanceapi.OpenSingleStream(streamName)
	if err != nil {
		log.WithError(err).
			WithField("stream", streamName).
			Errorf("Failed to open trade stream")
		time.Sleep(1 * time.Second)
		goto Retry
	}
	log.WithFields(log.Fields{
		"symbol": name,
	}).Infof("Connected to trade Binance aggTrade stream")
	for {
		payload, err := stream.Next()
		if err != nil {
			log.WithError(err).
				WithField("stream", streamName).
				Errorf("Failed to read trade stream message")
			stream.Close()
			time.Sleep(1 * time.Second)
			goto Retry
		}

		// Check if we still have subscribers.
		count := m.streamRefCount(name)
		if count == 0 {
			log.WithFields(log.Fields{
				"tickerStream": name,
			}).Infof("Trade stream reference count is zero, disconnected stream")
			stream.Close()
			return
		}

		var trade binanceapi.StreamAggTrade
		if err := json.Unmarshal(payload, &trade); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"name": name,
			}).Errorf("Failed to decode trade stream message")
			continue
		}

		m.lock.RLock()
		for channel := range m.subscriptions {
			select {
			case channel <- trade:
			default:
				log.Warnf("Failed to send Binance trade to channel [%s], would block",
					m.subscriptions[channel])
			}
		}
		m.lock.RUnlock()
	}
}
