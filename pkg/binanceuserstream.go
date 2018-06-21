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
	"time"
	"github.com/crankykernel/cryptotrader/binance"
	"github.com/gorilla/websocket"
	"strings"
	"encoding/json"
	"github.com/crankykernel/maker/pkg/config"
	"github.com/crankykernel/maker/pkg/log"
)

type StreamEventType string

const (
	EventTypeExecutionReport     StreamEventType = "executionReport"
	EventTypeOutboundAccountInfo StreamEventType = "outboundAccountInfo"
)

type UserStreamEvent struct {
	EventType           StreamEventType
	EventTime           time.Time
	OutboundAccountInfo binance.StreamOutboundAccountInfo
	ExecutionReport     binance.StreamExecutionReport
	Raw                 []byte
}

type BinanceUserDataStream struct {
	Subscribers map[chan *UserStreamEvent]bool
	lock        sync.RWMutex
}

func NewBinanceUserDataStream() *BinanceUserDataStream {
	return &BinanceUserDataStream{
		Subscribers: make(map[chan *UserStreamEvent]bool),
	}
}

func (b *BinanceUserDataStream) Subscribe() (chan *UserStreamEvent) {
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

func (b *BinanceUserDataStream) Run() {
	b.DoRun()
}

func (b *BinanceUserDataStream) DoRun() {
	intervalDuration := time.Minute
	intervalChannel := make(chan bool)
	lastPong := time.Now()
	configChannel := config.Subscribe()

	go func() {
		for {
			time.Sleep(intervalDuration)
			select {
			case intervalChannel <- true:
			default:
				log.Println("Failed to send OK to interval channel.")
			}
		}
	}()

	goto Start

Fail:
	select {
	case intervalChannel <- false:
	default:
	}
	time.Sleep(time.Second)

Start:
	apiKey := config.GetString("binance.api.key")

	if apiKey == "" {
		log.Infof("Binance API key not set. Waiting for configuration update.")
		<-configChannel
		goto Start
	}

	restClient := binance.NewAuthenticatedClient(
		config.GetString("binance.api.key"), "")

	for {

		// First we have to get the user stream listen key.
		listenKey, err := restClient.GetUserDataStream()
		if err != nil {
			log.Printf("error: failed to get user stream key: %v",
				err)
			goto Fail
		}

		userStream, err := binance.OpenSingleStream(listenKey)
		if err != nil {
			log.Printf("Failed to open user data stream: %v", err)
			goto Fail
		}

		log.Infof("Connected to Binance user stream websocket.");
		userStream.Conn.SetPongHandler(func(appData string) error {
			lastPong = time.Now()
			return nil
		})

		go func() {
			for {
				if time.Now().Sub(lastPong) > intervalDuration*2 {
					log.Printf("ERROR: Last user stream PONG received %v ago.", intervalDuration)
					userStream.Close()
					return
				}

				if err := userStream.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					log.Printf("ERROR: Failed to send user stream PING message: %v", err)
					userStream.Close()
					return
				}

				err := restClient.PutUserStreamKeepAlive(listenKey)
				if err != nil {
					log.Printf("ERROR: Failed to send user stream keep alive: %v", err)
					userStream.Close()
					return
				}

				select {
				case ok := <-intervalChannel:
					if !ok {
						log.Println("PING loop exiting.")
						return
					}
				case <-configChannel:
					log.Info("Received configuration update notification.")
					userStream.Close()
					return
				}

			}
		}()

		for {
			_, message, err := userStream.Next()
			if err != nil {
				log.Printf("error: user stream: %v", err)
				goto Fail
			}

			streamEvent := UserStreamEvent{}
			streamEvent.Raw = message

			switch {
			case strings.HasPrefix(string(message), `{"e":"executionReport",`):
				var orderUpdate binance.StreamExecutionReport
				if err := json.Unmarshal(message, &orderUpdate); err != nil {
					log.Printf("error: failed to decode executionReport event")
				}
				streamEvent.EventType = StreamEventType(orderUpdate.EventType)
				streamEvent.EventTime = time.Unix(0, orderUpdate.EventTimeMillis*int64(time.Millisecond))
				streamEvent.ExecutionReport = orderUpdate
			case strings.HasPrefix(string(message), `{"e":"outboundAccountInfo",`):
				if err := json.Unmarshal(message, &streamEvent.OutboundAccountInfo); err != nil {
					log.Printf("error: failed to decode outputAccountInfo event")
				}
				streamEvent.EventType = StreamEventType(streamEvent.OutboundAccountInfo.EventType)
				streamEvent.EventTime = time.Unix(0, streamEvent.OutboundAccountInfo.EventTimeMillis*int64(time.Millisecond))
			}

			for channel := range b.Subscribers {
				channel <- &streamEvent
			}
		}
	}
}
