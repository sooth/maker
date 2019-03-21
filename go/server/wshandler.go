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

import (
	"encoding/json"
	"github.com/crankykernel/binanceapi-go"
	"github.com/gorilla/websocket"
	"gitlab.com/crankykernel/maker/go/binanceex"
	"gitlab.com/crankykernel/maker/go/clientnotificationservice"
	"gitlab.com/crankykernel/maker/go/context"
	"gitlab.com/crankykernel/maker/go/healthservice"
	"gitlab.com/crankykernel/maker/go/log"
	"gitlab.com/crankykernel/maker/go/tradeservice"
	"gitlab.com/crankykernel/maker/go/types"
	"gitlab.com/crankykernel/maker/go/version"
	"net/http"
)

// This handler implements the read-only websocket that all clients connect
// to for state updates.
type UserWebSocketHandler struct {
	appContext          *context.ApplicationContext
	clientNoticeService *clientnotificationservice.Service
	healthService       *healthservice.Service
}

func NewUserWebSocketHandler(
	appContext *context.ApplicationContext,
	clientNoticeService *clientnotificationservice.Service,
	healthService *healthservice.Service) *UserWebSocketHandler {
	return &UserWebSocketHandler{
		appContext:          appContext,
		clientNoticeService: clientNoticeService,
		healthService:       healthService,
	}
}

func (h *UserWebSocketHandler) readLoop(ws *websocket.Conn, doneChannel chan bool) {
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"remoteAddr": ws.RemoteAddr(),
			}).Errorf("Error reading next message from client websocket")
			select {
			case doneChannel <- true:
			default:
			}
			break
		}
	}
	log.WithField("remoteAddr", ws.RemoteAddr()).Debug("Client websocket read-loop done")
}

func (h *UserWebSocketHandler) writeLoop(ws *websocket.Conn, writeChannel chan *MakerMessage) {
	for {
		message := <-writeChannel
		if message == nil {
			break
		}
		if err := h.WriteMessage(ws, message); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"remoteAddr": ws.RemoteAddr(),
			}).Errorf("Failed to write message to websocket client")
			ws.Close()
			break
		}
	}
	log.WithField("remoteAddr", ws.RemoteAddr()).Debug("Client websocket write-loop done")
}

func (h *UserWebSocketHandler) WriteMessage(ws *websocket.Conn, message *MakerMessage) error {
	buf, err := json.Marshal(message)
	if err != nil {
		log.WithError(err).Error("Failed to encode message to JSON.")
		return nil
	}
	return ws.WriteMessage(websocket.TextMessage, buf)
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

	if err := ws.WriteJSON(map[string]interface{}{
		"messageType":  MakerMessageTypeVersion,
		"version":      version.Version,
		"git_revision": version.GitRevision,
	}); err != nil {
		log.WithError(err).Errorf("Failed to send version message to client websocket")
		return
	}

	log.WithFields(log.Fields{
		"remoteAddr": r.RemoteAddr,
	}).Info("Client websocket connected")

	doneChannel := make(chan bool)
	tradeChannel := h.appContext.TradeService.Subscribe("wshandler")
	defer h.appContext.TradeService.Unsubscribe(tradeChannel)

	binanceTradeStreamChannel := h.appContext.BinanceTradeStreamManager.Subscribe("wshandler")
	defer h.appContext.BinanceTradeStreamManager.Unsubscribe(binanceTradeStreamChannel)

	binanceUserStreamChannel := h.appContext.BinanceUserDataStream.Subscribe("wshandler")
	defer h.appContext.BinanceUserDataStream.Unsubscribe(binanceUserStreamChannel)

	writeChannel := make(chan *MakerMessage, 128)

	go h.readLoop(ws, doneChannel)
	go h.writeLoop(ws, writeChannel)

	trades := h.appContext.TradeService.GetAllTrades()
	for _, trade := range trades {
		message := map[string]interface{}{
			"messageType": MakerMessageTypeTrade,
			"trade":       trade.State,
		}
		bytes, err := json.Marshal(message)
		if err != nil {
			log.Printf("error: failed to convert message to json: %v", err)
		} else {
			ws.WriteMessage(websocket.TextMessage, bytes)
		}
	}

	clientNoticeChannel := h.clientNoticeService.Subscribe()
	defer h.clientNoticeService.Unsubscribe(clientNoticeChannel)

	healthUpdateChannel := h.healthService.Subscribe()
	defer h.healthService.Unsubscribe(healthUpdateChannel)

Loop:
	for {
		var outboundMessage *MakerMessage = nil
		select {
		case <-doneChannel:
			log.WithFields(log.Fields{
				"remoteAddr": ws.RemoteAddr(),
			}).Debugf("Received read-loop done message, closing down websocket")
			select {
			case writeChannel <- nil:
			default:
			}
			break Loop
		case binanceUserEvent := <-binanceUserStreamChannel:
			switch binanceUserEvent.EventType {
			case binanceex.EventTypeExecutionReport:
				// Do nothing.
			case binanceex.EventTypeOutboundAccountInfo:
				outboundMessage = &MakerMessage{
					Type:                       MakerMessageTypeBinanceAccountInfo,
					BinanceOutboundAccountInfo: binanceUserEvent.OutboundAccountInfo,
				}
			default:
				log.WithFields(log.Fields{
					"eventType": binanceUserEvent.EventType,
				}).Info("Ignoring binance user stream event.")
			}
		case trade := <-binanceTradeStreamChannel:
			outboundMessage = &MakerMessage{
				Type:            MakerMessageTypeBinanceAggTrade,
				BinanceAggTrade: trade,
			}
		case trade := <-tradeChannel:
			switch trade.EventType {
			case tradeservice.TradeEventTypeUpdate:
				outboundMessage = &MakerMessage{
					Type:  MakerMessageTypeTrade,
					Trade: trade.TradeState,
				}
			case tradeservice.TradeEventTypeArchive:
				outboundMessage = &MakerMessage{
					Type:    MakerMessageTypeTradeArchived,
					TradeID: trade.TradeID,
				}
			default:
				log.Printf("ERROR: Unknown trade server event type: %s",
					trade.EventType)
			}
		case notice := <-clientNoticeChannel:
			outboundMessage = &MakerMessage{
				Type:   MakerMessageTypeNotice,
				Notice: notice,
			}
		case health := <-healthUpdateChannel:
			outboundMessage = &MakerMessage{
				Type:   MakerMessageTypeHealth,
				Health: health,
			}
		}

		if outboundMessage != nil {
			select {
			case writeChannel <- outboundMessage:
			default:
				log.WithFields(log.Fields{
					"remoteAddr": ws.RemoteAddr(),
				}).Errorf("Too many messages queued for websocket client, dropping")
				ws.Close()
				break Loop
			}
		}

	}

	log.WithFields(log.Fields{
		"remoteAddr": r.RemoteAddr,
	}).Info("Client websocket closed")
}

type MakerMessage struct {
	Type                       MakerMessageType                     `json:"messageType"`
	Trade                      types.TradeState                     `json:"trade,omitempty"`
	TradeID                    string                               `json:"tradeId,omitempty"`
	BinanceAggTrade            binanceapi.StreamAggTrade            `json:"binanceAggTrade,omitempty"`
	BinanceOutboundAccountInfo binanceapi.StreamOutboundAccountInfo `json:"binanceOutboundAccountInfo,omitempty"`
	Notice                     *clientnotificationservice.Notice    `json:"notice,omitempty"`
	Health                     *healthservice.State                 `json:"health,omitempty"`
}

type MakerMessageType string

const MakerMessageTypeVersion MakerMessageType = "version"
const MakerMessageTypeTradeArchived MakerMessageType = "tradeArchived"
const MakerMessageTypeTrade MakerMessageType = "trade"
const MakerMessageTypeBinanceAggTrade MakerMessageType = "binanceAggTrade"
const MakerMessageTypeBinanceAccountInfo MakerMessageType = "binanceOutboundAccountInfo"
const MakerMessageTypeNotice MakerMessageType = "notice"
const MakerMessageTypeHealth MakerMessageType = "health"
