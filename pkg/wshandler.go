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
	"net/http"
	"github.com/gorilla/websocket"
	"encoding/json"
	"github.com/crankykernel/maker/pkg/log"
)

// This handler implements the read-only websocket that all clients connect
// to for state updates.
type UserWebSocketHandler struct {
	appContext *ApplicationContext
}

func NewUserWebSocketHandler(appContext *ApplicationContext) *UserWebSocketHandler {
	return &UserWebSocketHandler{
		appContext: appContext,
	}
}

func (h *UserWebSocketHandler) readLoop(ws *websocket.Conn, doneChannel chan bool) {
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			doneChannel <- true
			close(doneChannel)
			break
		}
	}
	log.Debugf("User WebSocket readLoop exiting.")
}

func (h *UserWebSocketHandler) writeLoop(ws *websocket.Conn, writeChannel chan interface{}) {
	for {
		message := <-writeChannel
		if message == nil {
			break
		}
		buf, err := json.Marshal(message)
		if err != nil {
			log.WithError(err).Error("Failed to encode message to JSON.")
			continue
		}
		if err := ws.WriteMessage(websocket.TextMessage, buf); err != nil {
			log.Printf("error: failed to write to websocket: %v", err)
			return
		}
	}
	log.Debugf("User WebSocket writeLoop exiting.")
}

func (h *UserWebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error: failed to upgrade to websocket: %v", err)
		return
	}

	defer func() {
		ws.Close()
	}()

	doneChannel := make(chan bool)
	tradeChannel := h.appContext.TradeService.Subscribe()
	defer h.appContext.TradeService.Unsubscribe(tradeChannel)

	binanceTradeStreamChannel := h.appContext.BinanceStreamManager.SubscribeTrades()
	defer h.appContext.BinanceStreamManager.UnsubscribeTrades(binanceTradeStreamChannel)

	writeChannel := make(chan interface{})

	go h.readLoop(ws, doneChannel)
	go h.writeLoop(ws, writeChannel)

	trades := h.appContext.TradeService.GetAllTrades()
	for _, trade := range trades {
		message := map[string]interface{}{
			"messageType": "trade",
			"trade":       trade.State,
		}
		bytes, err := json.Marshal(message)
		if err != nil {
			log.Printf("error: failed to convert message to json: %v", err)
		} else {
			ws.WriteMessage(websocket.TextMessage, bytes)
		}
	}

Loop:
	for {
		select {
		case <-doneChannel:
			break Loop
		case trade := <-binanceTradeStreamChannel:
			message := map[string]interface{}{
				"messageType":     "binanceAggTrade",
				"binanceAggTrade": trade,
			}
			writeChannel <- message
		case trade := <-tradeChannel:
			message := map[string]interface{}{}

			switch trade.EventType {
			case TradeEventTypeUpdate:
				message["messageType"] = "trade"
				message["trade"] = trade.TradeState
			case TradeEventTypeArchive:
				message["messageType"] = "tradeArchived"
				message["tradeId"] = trade.TradeID
			default:
				log.Printf("ERROR: Unknown trade server event type: %s",
					trade.EventType)
				continue
			}
			writeChannel <- message
		}
	}

	writeChannel <- nil

	log.Debugf("User WebSocket closed.")
}
