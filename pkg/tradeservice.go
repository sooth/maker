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
	"github.com/crankykernel/cryptotrader/binance"
	"sync"
	"math"
	"fmt"
	"github.com/crankykernel/maker/pkg/log"
	"time"
	"github.com/crankykernel/maker/pkg/idgenerator"
	"github.com/crankykernel/maker/pkg/maker"
	"github.com/crankykernel/maker/pkg/db"
)

type TradeEventType string

const (
	TradeEventTypeUpdate  TradeEventType = "update"
	TradeEventTypeArchive TradeEventType = "archived"
)

type TradeEvent struct {
	EventType  TradeEventType
	TradeState *maker.TradeState
	TradeID    string
}

type TradeService struct {
	// TradesByClientID by local ID. This map should only contain a single instance of
	// each trade.
	TradesByLocalID map[string]*maker.Trade

	// TradesByClientID by client ID. This will contain multiple instances of the same
	// trade as a key is created for each client ID associated with the trade.
	TradesByClientID map[string]*maker.Trade

	idGenerator *idgenerator.IdGenerator

	subscribers map[chan TradeEvent]bool
	lock        sync.RWMutex

	binanceStreamManager *BinanceStreamManager
	applicationContext   *ApplicationContext
	tradeStreamChannel   chan *binance.StreamAggTrade

	binanceExchangeInfo *binance.ExchangeInfoService
}

func NewTradeService(applicationContext *ApplicationContext) *TradeService {
	tradeService := &TradeService{
		TradesByLocalID:      make(map[string]*maker.Trade),
		TradesByClientID:     make(map[string]*maker.Trade),
		idGenerator:          idgenerator.NewIdGenerator(),
		subscribers:          make(map[chan TradeEvent]bool),
		applicationContext:   applicationContext,
		binanceStreamManager: applicationContext.BinanceStreamManager,
		binanceExchangeInfo:  binance.NewExchangeInfoService(),
	}

	if err := tradeService.binanceExchangeInfo.Update(); err != nil {
		log.Printf("error: failed to update exchange info: %v", err)
	}

	tradeService.tradeStreamChannel = tradeService.binanceStreamManager.SubscribeTrades()
	go tradeService.tradeStreamListener()

	return tradeService
}

func (s *TradeService) tradeStreamListener() {
	for {
		lastTrade := <-s.tradeStreamChannel
		s.onLastTrade(lastTrade)
	}
}

func (s *TradeService) onLastTrade(lastTrade *binance.StreamAggTrade) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, trade := range s.TradesByLocalID {

		if trade.IsDone() {
			continue
		}

		if trade.State.Symbol == lastTrade.Symbol {

			switch trade.State.Status {
			case maker.TradeStatusPendingSell:
			case maker.TradeStatusWatching:
			default:
				continue
			}

			trade.State.LastPrice = lastTrade.Price

			lastEffectivePrice := lastTrade.Price * (1 - trade.State.Fee)
			trade.State.ProfitPercent = (lastEffectivePrice - trade.State.EffectiveBuyPrice) /
				trade.State.EffectiveBuyPrice * 100

			if trade.State.StopLoss.Enabled {
				s.checkStopLoss(trade, lastTrade)
			}
			if trade.State.TrailingStop.Enabled {
				s.checkTrailingStop(trade, lastTrade)
			}
		}
	}
}

func (s *TradeService) checkStopLoss(trade *maker.Trade, lastTrade *binance.StreamAggTrade) {
	switch trade.State.Status {
	case maker.TradeStatusPendingSell:
	case maker.TradeStatusWatching:
	default:
		return
	}
	if trade.State.StopLoss.Triggered {
		return
	}
	if trade.State.ProfitPercent < math.Abs(trade.State.StopLoss.Percent) * -1 {
		log.WithFields(log.Fields{
			"symbol": trade.State.Symbol,
			"loss":   trade.State.ProfitPercent,
		}).Infof("Stop Loss: Triggering market sell.")
		if trade.State.Status == maker.TradeStatusPendingSell {
			s.CancelSell(trade)
		}
		trade.State.StopLoss.Triggered = true
		s.MarketSell(trade, true)
	}
}

func (s *TradeService) checkTrailingStop(trade *maker.Trade, lastTrade *binance.StreamAggTrade) {
	switch trade.State.Status {
	case maker.TradeStatusPendingSell:
	case maker.TradeStatusWatching:
	default:
		return
	}

	if trade.State.TrailingStop.Triggered {
		return
	}

	if trade.State.TrailingStop.Activated {
		if lastTrade.Price > trade.State.TrailingStop.Price {
			trade.State.TrailingStop.Price = lastTrade.Price
			log.WithFields(log.Fields{
				"symbol":   trade.State.Symbol,
				"price-hi": lastTrade.Price,
			}).Info("Trailing Stop: Increasing high price.")
		} else {
			deviation := (lastTrade.Price - trade.State.TrailingStop.Price) /
				trade.State.TrailingStop.Price * 100
			log.Printf("%s: TrailingStop Deviation: deviation=%.8f; allowed=%.8f",
				trade.State.Symbol, deviation, trade.State.TrailingStop.Deviation)
			if math.Abs(deviation) > trade.State.TrailingStop.Deviation {
				log.Printf("%s: TrailingStop: Triggering sell: profit=%.8f",
					trade.State.Symbol, trade.State.ProfitPercent)
				trade.State.TrailingStop.Triggered = true
				s.MarketSell(trade, true)
			}
		}
	} else {
		if trade.State.ProfitPercent > trade.State.TrailingStop.Percent {
			log.Printf("%s: Activating trailing stop: profit=%.8f; percent: %.8f",
				trade.State.Symbol, trade.State.ProfitPercent,
				trade.State.TrailingStop.Percent)
			trade.State.TrailingStop.Activated = true
			s.BroadcastTradeUpdate(trade)
		}
	}

}

func (s *TradeService) GetAllTrades() []*maker.Trade {
	s.lock.RLock()
	defer s.lock.RUnlock()
	trades := []*maker.Trade{}
	for _, trade := range s.TradesByLocalID {
		trades = append(trades, trade)
	}
	return trades
}

func (s *TradeService) Subscribe() chan TradeEvent {
	s.lock.Lock()
	defer s.lock.Unlock()
	channel := make(chan TradeEvent)
	s.subscribers[channel] = true
	return channel
}

func (s *TradeService) Unsubscribe(channel chan TradeEvent) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.subscribers[channel] = false;
	delete(s.subscribers, channel)
}

func (s *TradeService) broadcastTradeEvent(tradeEvent TradeEvent) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for channel := range s.subscribers {
		channel <- tradeEvent
	}
}

func (s *TradeService) BroadcastTradeUpdate(trade *maker.Trade) {
	tradeEvent := TradeEvent{
		EventType:  TradeEventTypeUpdate,
		TradeState: &trade.State,
	}
	s.broadcastTradeEvent(tradeEvent)
}

func (s *TradeService) BroadcastTradeArchived(tradeId string) {
	tradeEvent := TradeEvent{
		EventType: TradeEventTypeArchive,
		TradeID:   tradeId,
	}
	s.broadcastTradeEvent(tradeEvent)
}

func (s *TradeService) FindTradeByLocalID(localId string) *maker.Trade {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.TradesByLocalID[localId]
}

func (s *TradeService) AbandonTrade(trade *maker.Trade) {
	if !trade.IsDone() {
		s.CloseTrade(trade, maker.TradeStatusAbandoned, time.Now())
		s.BroadcastTradeUpdate(trade)
	}
}

func (s *TradeService) ArchiveTrade(trade *maker.Trade) error {
	if trade.IsDone() {
		if err := db.DbArchiveTrade(trade); err != nil {
			return err
		}
		s.RemoveTrade(trade)
		s.BroadcastTradeArchived(trade.State.LocalID)
		return nil
	}
	return fmt.Errorf("archive not allowed in state %s", trade.State.Status)
}

func (s *TradeService) AddClientOrderId(trade *maker.Trade, orderId string, locked bool) {
	log.Printf("Adding clientOrderID %s to trade %s", orderId, trade.State.LocalID)
	trade.AddClientOrderID(orderId)
	if !locked {
		s.lock.Lock()
	}
	s.TradesByClientID[orderId] = trade
	if !locked {
		s.lock.Unlock()
	}
	db.DbUpdateTrade(trade)
}

func (s *TradeService) RestoreTrade(trade *maker.Trade) {
	s.TradesByLocalID[trade.State.LocalID] = trade
	for clientOrderId := range trade.State.ClientOrderIDs {
		s.TradesByClientID[clientOrderId] = trade
	}
	s.binanceStreamManager.SubscribeTradeStream(trade.State.Symbol)
}

func (s *TradeService) AddNewTrade(trade *maker.Trade) (string) {
	if trade.State.LocalID == "" {
		localId, err := s.idGenerator.GetID(nil)
		if err != nil {
			log.Fatalf("error: failed to generate trade id: %v", err)
		}
		trade.State.LocalID = localId.String()
	}
	trade.State.Status = maker.TradeStatusNew

	s.lock.Lock()
	s.TradesByLocalID[trade.State.LocalID] = trade
	for clientOrderId := range trade.State.ClientOrderIDs {
		log.WithFields(log.Fields{
			"tradeId":       trade.State.LocalID,
			"clientOrderId": clientOrderId,
		}).Debugf("Recording clientOrderId for new trade.")
		s.TradesByClientID[clientOrderId] = trade
	}
	s.lock.Unlock()

	if err := db.DbSaveTrade(trade); err != nil {
		log.Printf("error: failed to save trade to database: %v", err)
	}

	s.binanceStreamManager.SubscribeTradeStream(trade.State.Symbol)
	s.BroadcastTradeUpdate(trade)

	return trade.State.LocalID
}

func (s *TradeService) RemoveTrade(trade *maker.Trade) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.TradesByLocalID, trade.State.LocalID)
	for clientId := range trade.State.ClientOrderIDs {
		log.WithFields(log.Fields{
			"tradeId":       trade.State.LocalID,
			"clientOrderId": clientId,
		}).Debugf("Removing trade from clientOrderId map.")
		delete(s.TradesByClientID, clientId)
	}
}

func (s *TradeService) FailTrade(trade *maker.Trade) {
	s.CloseTrade(trade, maker.TradeStatusFailed, time.Now())
	s.BroadcastTradeUpdate(trade)
}

func (s *TradeService) FindTradeForReport(report binance.StreamExecutionReport) *maker.Trade {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if trade, ok := s.TradesByClientID[report.ClientOrderID]; ok {
		log.WithFields(log.Fields{
			"clientOrderId": report.ClientOrderID,
		}).Debugf("Found trade by clientOrderId.")
		return trade
	}

	log.WithFields(log.Fields{
		"clientOrderId":     report.ClientOrderID,
		"origClientOrderId": report.OriginalClientOrderID,
	}).Debugf("Failed to find trade by clientOrderId, trying origClientOrderId.")

	if trade, ok := s.TradesByClientID[report.OriginalClientOrderID]; ok {
		log.WithFields(log.Fields{
			"clientOrderId":     report.ClientOrderID,
			"origClientOrderId": report.OriginalClientOrderID,
		}).Debugf("Found trade by origClientOrderId.")
		return trade
	}

	log.WithFields(log.Fields{
		"clientOrderId":     report.ClientOrderID,
		"origClientOrderId": report.OriginalClientOrderID,
	}).Debugf("Failed to find trade by origClientorderId.")

	return nil
}

func (s *TradeService) OnExecutionReport(event *UserStreamEvent) {
	report := event.ExecutionReport

	trade := s.FindTradeForReport(report)
	if trade == nil {
		log.Errorf("Failed to find trade for execution report: %s", log.ToJson(report))
		return
	}

	if trade.IsDone() {
		return
	}

	log.WithFields(log.Fields{
		"tradeId": trade.State.LocalID,
	}).Debugf("Received execution report: %s", log.ToJson(report))

	switch report.Side {
	case binance.OrderSideBuy:
		switch report.CurrentOrderStatus {
		case binance.OrderStatusNew:
			trade.State.OpenTime = event.EventTime
			trade.State.BuyOrder.Quantity = report.Quantity
			trade.State.BuyOrder.Price = report.Price
			trade.State.BuyOrderId = report.OrderID
			if trade.State.Status == maker.TradeStatusNew {
				trade.State.Status = maker.TradeStatusPendingBuy
			}
			if trade.State.LastBuyStatus == "" {
				trade.State.LastBuyStatus = report.CurrentOrderStatus
			}
		case binance.OrderStatusCanceled:
			if trade.State.BuyFillQuantity == 0 {
				trade.State.Status = maker.TradeStatusCanceled
			} else {
				trade.State.Status = maker.TradeStatusWatching
			}
			trade.State.LastBuyStatus = report.CurrentOrderStatus
		case binance.OrderStatusPartiallyFilled:
			if trade.State.LastBuyStatus != binance.OrderStatusFilled {
				trade.State.LastBuyStatus = report.CurrentOrderStatus
			}
			trade.AddBuyFill(report)
		case binance.OrderStatusFilled:
			trade.AddBuyFill(report)
			trade.State.Status = maker.TradeStatusWatching
			if trade.State.LimitSell.Enabled {
				s.TriggerLimitSell(trade)
			}
			trade.State.LastBuyStatus = report.CurrentOrderStatus
		}
	case binance.OrderSideSell:
		switch report.CurrentOrderStatus {
		case binance.OrderStatusNew:
			if trade.State.Status == maker.TradeStatusDone {
				// Sometimes we get the fill before the new.
				break
			}
			trade.State.SellOrderId = report.OrderID
			trade.State.Status = maker.TradeStatusPendingSell
			switch trade.State.SellOrder.Status {
			case binance.OrderStatusPartiallyFilled:
			case binance.OrderStatusFilled:
			default:
				trade.State.SellOrder.Status = report.CurrentOrderStatus
			}
			trade.State.SellOrder.Type = report.OrderType
			trade.State.SellOrder.Quantity = report.Quantity
			trade.State.SellOrder.Price = report.Price
		case binance.OrderStatusPartiallyFilled:
			fill := maker.OrderFill{
				Price:            report.LastExecutedPrice,
				Quantity:         report.LastExecutedQuantity,
				CommissionAsset:  report.CommissionAsset,
				CommissionAmount: report.CommissionAmount,
			}
			trade.DoAddSellFill(fill)
			if trade.State.SellOrder.Status != binance.OrderStatusFilled {
				trade.State.SellOrder.Status = report.CurrentOrderStatus
			}
		case binance.OrderStatusFilled:
			fill := maker.OrderFill{
				Price:            report.LastExecutedPrice,
				Quantity:         report.LastExecutedQuantity,
				CommissionAsset:  report.CommissionAsset,
				CommissionAmount: report.CommissionAmount,
			}
			trade.DoAddSellFill(fill)
			trade.State.Status = maker.TradeStatusDone
			trade.State.SellOrder.Status = report.CurrentOrderStatus
		case binance.OrderStatusCanceled:
			trade.State.Status = maker.TradeStatusWatching
			trade.State.SellOrder.Status = report.CurrentOrderStatus
		default:
			log.Printf("ERROR: Unhandled order status state %s for sell side.",
				report.CurrentOrderStatus)
			trade.State.SellOrder.Status = report.CurrentOrderStatus
		}
	}

	switch trade.State.Status {
	case maker.TradeStatusDone:
		fallthrough
	case maker.TradeStatusCanceled:
		fallthrough
	case maker.TradeStatusFailed:
		trade.State.CloseTime = &event.EventTime
		s.binanceStreamManager.UnsubscribeTradeStream(trade.State.Symbol)
		fallthrough
	default:
		db.DbUpdateTrade(trade)
	}

	s.BroadcastTradeUpdate(trade)
}

func (s *TradeService) CloseTrade(trade *maker.Trade, status maker.TradeStatus, closeTime time.Time) {
	if closeTime.IsZero() {
		closeTime = time.Now()
	}
	trade.State.Status = status
	trade.State.CloseTime = &closeTime
	s.binanceStreamManager.UnsubscribeTradeStream(trade.State.Symbol)
	db.DbUpdateTrade(trade)
}

func (s *TradeService) TriggerLimitSell(trade *maker.Trade) {
	s.DoLimitSell(trade, trade.State.LimitSell.Percent)
}

func (s *TradeService) DoLimitSell(trade *maker.Trade, percent float64) error {
	tickSize, err := s.binanceExchangeInfo.GetTickSize(trade.State.Symbol)
	if err != nil {
		log.Printf("ERROR: Failed to get tick size for %s: %v: Limit sell not posted.", err)
		return err
	}

	price := trade.State.EffectiveBuyPrice * (1 + (percent / 100)) * (1 + trade.State.Fee)
	fixedPrice := roundx(price, 1/tickSize)

	clientOrderId, err := s.MakeOrderID()
	if err != nil {
		log.Printf("ERROR: Failed to generate clientOrderId: %v", err)
		return err
	}
	s.AddClientOrderId(trade, clientOrderId, false)

	log.WithFields(log.Fields{
		"limitPrice": price,
		"fixedPrice": fixedPrice,
		"symbol":     trade.State.Symbol,
		"tradeId":    trade.State.LocalID,
	}).Debugf("Posting sell order.")

	order := binance.OrderParameters{
		Symbol:           trade.State.Symbol,
		Side:             binance.OrderSideSell,
		Type:             binance.OrderTypeLimit,
		TimeInForce:      binance.TimeInForceGTC,
		Quantity:         trade.State.BuyFillQuantity,
		Price:            fixedPrice,
		NewClientOrderId: clientOrderId,
	}
	s0 := time.Now()
	_, err = getBinanceRestClient().PostOrder(order)
	d := time.Now().Sub(s0)
	if err != nil {
		log.WithFields(log.Fields{
			"requestDuration": d,
		}).WithError(err).Error("Failed to send sell order.")
		return err
	}
	log.WithFields(log.Fields{
		"requestDuration": d,
		"limitPrice":      price,
		"fixedPrice":      fixedPrice,
		"symbol":          trade.State.Symbol,
		"tradeId":         trade.State.LocalID,
	}).Info("Sell order posted.")
	db.DbUpdateTrade(trade)
	return nil
}

func (s *TradeService) UpdateStopLoss(trade *maker.Trade, enable bool, percent float64) {
	trade.SetStopLoss(enable, percent)
	db.DbUpdateTrade(trade)
	s.BroadcastTradeUpdate(trade)
}

func (s *TradeService) MakeOrderID() (string, error) {
	now := time.Now()
	orderId, err := s.idGenerator.GetID(&now)
	if err != nil {
		return "", err
	}
	return orderId.String(), nil
}

func (s *TradeService) UpdateTrailingStop(trade *maker.Trade, enable bool,
	percent float64, deviation float64) {
	trade.SetTrailingStop(enable, percent, deviation)
	db.DbUpdateTrade(trade)
	s.BroadcastTradeUpdate(trade)
}

func (s *TradeService) CancelSell(trade *maker.Trade) error {
	log.Printf("Cancelling sell order: symbol=%s; orderId=%d",
		trade.State.Symbol, trade.State.SellOrderId)
	_, err := getBinanceRestClient().CancelOrder(
		trade.State.Symbol, trade.State.SellOrderId)
	return err
}

func (s *TradeService) MarketSell(trade *maker.Trade, locked bool) error {
	clientOrderId, err := s.MakeOrderID()
	if err != nil {
		log.WithError(err).Errorf("Failed to generate order ID")
		return err
	}

	s.AddClientOrderId(trade, clientOrderId, locked)

	order := binance.OrderParameters{
		Symbol:           trade.State.Symbol,
		Side:             binance.OrderSideSell,
		Type:             binance.OrderTypeMarket,
		Quantity:         trade.State.BuyFillQuantity,
		NewClientOrderId: clientOrderId,
	}
	_, err = getBinanceRestClient().PostOrder(order)
	if err == nil {
		db.DbUpdateTrade(trade)
	}
	return err
}

func roundx(val float64, x float64) float64 {
	return math.Round(val*x) / x
}
