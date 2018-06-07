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

package pkg

import (
	"sync"
	"github.com/crankykernel/cryptotrader/binance"
	"strings"
	"github.com/crankykernel/maker/pkg/log"
)

type BinanceStreamManager struct {
	fatLock                        sync.Mutex
	tradeStreamCount               map[string]uint
	tradeStreamUnsubscribeChannels map[string]chan bool
	tradeStreamSubscriptions       map[chan *binance.StreamAggTrade]bool
}

func NewBinanceStreamManager() *BinanceStreamManager {
	return &BinanceStreamManager{
		tradeStreamCount:               make(map[string]uint),
		tradeStreamSubscriptions:       make(map[chan *binance.StreamAggTrade]bool),
		tradeStreamUnsubscribeChannels: make(map[string]chan bool),
	}
}

func (m *BinanceStreamManager) SubscribeTrades() chan *binance.StreamAggTrade {
	m.fatLock.Lock()
	defer m.fatLock.Unlock()
	channel := make(chan *binance.StreamAggTrade)
	m.tradeStreamSubscriptions[channel] = true
	return channel
}

func (m *BinanceStreamManager) UnsubscribeTrades(channel chan *binance.StreamAggTrade) {
	m.fatLock.Lock()
	defer m.fatLock.Unlock()
	m.tradeStreamSubscriptions[channel] = false
	delete(m.tradeStreamSubscriptions, channel)
}

func (m *BinanceStreamManager) SubscribeTradeStream(symbol string) {
	symbol = strings.ToLower(symbol)
	m.fatLock.Lock()
	defer m.fatLock.Unlock()
	count, exists := m.tradeStreamCount[symbol]
	if exists && count > 0 {
		log.WithFields(log.Fields{
			"symbol": symbol,
			"count": count,
		}).Infof("Trade stream already exists.")
		m.tradeStreamCount[symbol] += 1
		return
	}
	m.tradeStreamCount[symbol] = 1
	go m.RunTradeStream(symbol)
}

func (m *BinanceStreamManager) UnsubscribeTradeStream(symbol string) {
	symbol = strings.ToLower(symbol)
	m.fatLock.Lock()
	defer m.fatLock.Unlock()
	count, exists := m.tradeStreamCount[symbol]
	if !exists {
		return
	}
	m.tradeStreamCount[symbol] = count - 1
	if m.tradeStreamCount[symbol] == 0 {
		m.tradeStreamUnsubscribeChannels[symbol] <- true
	}
}

func (m *BinanceStreamManager) RunTradeStream(symbol string) {
	for {
		log.Printf("Opening trade stream for %s.", symbol)
		streamClient, err := binance.OpenAggTradeStream(symbol)
		if err != nil {
			log.Printf("failed to open aggTrade stream: %v", err)
			return
		}

		m.fatLock.Lock()
		unsubscribeChannel := make(chan bool)
		m.tradeStreamUnsubscribeChannels[symbol] = unsubscribeChannel
		m.fatLock.Unlock()
		channel := make(chan binance.AggTradeStreamEvent)
		go streamClient.Subscribe(channel)

		for {
			select {
			case <-unsubscribeChannel:
				log.Printf("Closing trade subscription for %s.", symbol)
				streamClient.Close()
				return
			case event := <-channel:
				if event.Err != nil {
					log.Printf("failed to read from aggTrade stream: %v", event.Err)
					return
				}
				m.fatLock.Lock()
				for channel := range m.tradeStreamSubscriptions {
					channel <- event.Trade
				}
				m.fatLock.Unlock()
			}
		}
	}
}
