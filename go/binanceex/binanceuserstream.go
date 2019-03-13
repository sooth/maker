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
	"github.com/crankykernel/binanceapi-go"
	"gitlab.com/crankykernel/maker/go/clientnotificationservice"
	"gitlab.com/crankykernel/maker/go/config"
	"gitlab.com/crankykernel/maker/go/healthservice"
	"gitlab.com/crankykernel/maker/go/log"
	"strings"
	"sync"
	"time"
)

type StreamEventType string

const (
	EventTypeExecutionReport     StreamEventType = "executionReport"
	EventTypeOutboundAccountInfo StreamEventType = "outboundAccountInfo"
)

type ListenKeyWrapper struct {
	lock      sync.Mutex
	listenKey string
}

func NewListenKeyWrapper() *ListenKeyWrapper {
	return &ListenKeyWrapper{}
}

func (k *ListenKeyWrapper) Set(listenKey string) {
	k.lock.Lock()
	defer k.lock.Unlock()
	k.listenKey = listenKey
}

func (k *ListenKeyWrapper) Get() string {
	k.lock.Lock()
	defer k.lock.Unlock()
	return k.listenKey
}

type UserStreamEvent struct {
	EventType           StreamEventType
	EventTime           time.Time
	OutboundAccountInfo binanceapi.StreamOutboundAccountInfo
	ExecutionReport     binanceapi.StreamExecutionReport
	Raw                 []byte
}

type BinanceUserDataStream struct {
	Subscribers         map[chan *UserStreamEvent]bool
	lock                sync.RWMutex
	listenKey           *ListenKeyWrapper
	notificationService *clientnotificationservice.Service
	healthService       *healthservice.Service
}

func NewBinanceUserDataStream(notificationService *clientnotificationservice.Service,
	healthService *healthservice.Service) *BinanceUserDataStream {
	return &BinanceUserDataStream{
		Subscribers:         make(map[chan *UserStreamEvent]bool),
		listenKey:           NewListenKeyWrapper(),
		notificationService: notificationService,
		healthService:       healthService,
	}
}

func (b *BinanceUserDataStream) Subscribe() chan *UserStreamEvent {
	b.lock.Lock()
	defer b.lock.Unlock()
	channel := make(chan *UserStreamEvent)
	b.Subscribers[channel] = true
	return channel
}

func (b *BinanceUserDataStream) Unsubscribe(channel chan *UserStreamEvent) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.Subscribers[channel] = false
	delete(b.Subscribers, channel)
}

func (b *BinanceUserDataStream) ListenKeyRefreshLoop() {
	for {
		time.Sleep(time.Minute)
		listenKey := b.listenKey.Get()
		if listenKey == "" {
			log.Debugf("No Binance user stream key set, will not refresh")
		} else {
			log.Debugf("Refreshing Binance user stream listen key")
			client := GetBinanceRestClient()
			if err := client.PutUserStreamKeepAlive(listenKey); err != nil {
				log.WithError(err).Errorf("Failed to send Binance user stream keep alive.")
			}
		}
	}
}

func (b *BinanceUserDataStream) Run() {
	lastPong := time.Now()
	configChannel := config.Subscribe()

	go b.ListenKeyRefreshLoop()

	goto Start
Fail:
	b.listenKey.Set("")
	b.notificationService.Broadcast(
		clientnotificationservice.NewNotice(
			clientnotificationservice.LevelError,
			"Failed to connect to Binance user socket").
			WithData(map[string]interface{}{
				"binanceUserSocketState": "failed",
			}))
	b.healthService.Update(func(state *healthservice.State) {
		state.BinanceUserSocketState = "connection failed"
	})
	time.Sleep(time.Second)
Start:
	apiKey := config.GetString("binance.api.key")

	// Wait for key to be set if needed.
	if apiKey == "" {
		log.Infof("Binance API key not set. Waiting for configuration update.")
		<-configChannel
		goto Start
	}

	// First we have to get the user stream listen key.
	listenKey, err := GetBinanceRestClient().GetUserDataStream()
	if err != nil {
		log.WithError(err).Error("Failed to get Binance user stream key. Retyring.")
		goto Fail
	} else {
		log.WithFields(log.Fields{
		}).Debugf("Acquired Binance user stream listen key")
	}

	userStream, err := binanceapi.OpenSingleStream(listenKey)
	if err != nil {
		log.WithError(err).Errorf("Failed to open Binance user stream")
		goto Fail
	}
	b.listenKey.Set(listenKey)

	log.Infof("Connected to Binance user stream websocket.")
	userStream.Conn.SetPongHandler(func(appData string) error {
		log.WithFields(log.Fields{
			"data": appData,
		}).Debugf("Received Binance user stream pong")
		lastPong = time.Now()
		return nil
	})
	b.notificationService.Broadcast(
		clientnotificationservice.NewNotice(
			clientnotificationservice.LevelInfo,
			"Connected to Binance user data stream.").WithData(map[string]interface{}{
			"binanceUserSocketState": "ok",
		}))
	b.healthService.Update(func(state *healthservice.State) {
		state.BinanceUserSocketState = "ok"
	})

	userStream.Conn.SetPingHandler(func(appData string) error {
		log.WithFields(log.Fields{
			"data": appData,
		}).Debugf("Received Binance user stream ping")
		return nil
	})

	for {
		message, err := userStream.Next()
		if err != nil {
			log.WithError(err).Errorf("Failed to read next Binance user stream message")
			goto Fail
		}

		streamEvent := UserStreamEvent{}
		streamEvent.Raw = message

		switch {
		case strings.HasPrefix(string(message), `{"e":"executionReport",`):
			var orderUpdate binanceapi.StreamExecutionReport
			if err := json.Unmarshal(message, &orderUpdate); err != nil {
				log.WithError(err).Error("Failed to decode user stream executionReport message.")
				continue
			}
			streamEvent.EventType = StreamEventType(orderUpdate.EventType)
			streamEvent.EventTime = time.Unix(0, orderUpdate.EventTimeMillis*int64(time.Millisecond))
			streamEvent.ExecutionReport = orderUpdate
		case strings.HasPrefix(string(message), `{"e":"outboundAccountInfo",`):
			if err := json.Unmarshal(message, &streamEvent.OutboundAccountInfo); err != nil {
				log.WithError(err).Error("Failed to decode user stream outboundAccountInfo message.")
				continue
			}
			streamEvent.EventType = StreamEventType(streamEvent.OutboundAccountInfo.EventType)
			streamEvent.EventTime = time.Unix(0, streamEvent.OutboundAccountInfo.EventTimeMillis*int64(time.Millisecond))
		}

		for channel := range b.Subscribers {
			channel <- &streamEvent
		}
	}
}
