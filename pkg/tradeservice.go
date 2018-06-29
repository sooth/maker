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
	"gitlab.com/crankykernel/cryptotrader/binance"
	"sync"
	"math"
	"fmt"
	"gitlab.com/crankykernel/maker/pkg/log"
	"time"
	"gitlab.com/crankykernel/maker/pkg/idgenerator"
	"gitlab.com/crankykernel/maker/pkg/maker"
	"gitlab.com/crankykernel/maker/pkg/db"
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

// Calculate the profit based on the trade being sold at the given price.
// Returns a percentage value in the range of 0-100.
func (s *TradeService) CalculateProfit(trade *maker.Trade, price float64) float64 {
	grossSellCost := price * trade.State.SellableQuantity
	netSellCost := grossSellCost * (1 - trade.State.Fee)
	profit := (netSellCost - trade.State.BuyCost) / trade.State.BuyCost * 100
	return profit
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
			trade.State.ProfitPercent = s.CalculateProfit(trade, lastTrade.Price)

			if trade.State.StopLoss.Enabled {
				s.checkStopLoss(trade)
			}
			if trade.State.TrailingProfit.Enabled {
				s.checkTrailingProfit(trade, lastTrade.Price)
			}
		}
	}
}

func (s *TradeService) checkStopLoss(trade *maker.Trade) {
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

func (s *TradeService) checkTrailingProfit(trade *maker.Trade, price float64) {
	switch trade.State.Status {
	case maker.TradeStatusPendingSell:
	case maker.TradeStatusWatching:
	default:
		return
	}

	if trade.State.TrailingProfit.Triggered {
		return
	}

	if trade.State.TrailingProfit.Activated {
		if price > trade.State.TrailingProfit.Price {
			trade.State.TrailingProfit.Price = price
			log.WithFields(log.Fields{
				"symbol":   trade.State.Symbol,
				"price-hi": price,
			}).Info("Trailing Stop: Increasing high price.")
		} else {
			deviation := (price - trade.State.TrailingProfit.Price) /
				trade.State.TrailingProfit.Price * 100
			log.Printf("%s: TrailingProfit Deviation: deviation=%.8f; allowed=%.8f",
				trade.State.Symbol, deviation, trade.State.TrailingProfit.Deviation)
			if math.Abs(deviation) > trade.State.TrailingProfit.Deviation {
				log.Printf("%s: TrailingProfit: Triggering sell: profit=%.8f",
					trade.State.Symbol, trade.State.ProfitPercent)
				trade.State.TrailingProfit.Triggered = true
				s.MarketSell(trade, true)
			}
		}
	} else {
		if trade.State.ProfitPercent > trade.State.TrailingProfit.Percent {
			log.Printf("%s: Activating trailing stop: profit=%.8f; percent: %.8f",
				trade.State.Symbol, trade.State.ProfitPercent,
				trade.State.TrailingProfit.Percent)
			trade.State.TrailingProfit.Activated = true
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
		s.BroadcastTradeArchived(trade.State.TradeID)
		return nil
	}
	return fmt.Errorf("archive not allowed in state %s", trade.State.Status)
}

func (s *TradeService) AddClientOrderId(trade *maker.Trade, orderId string, locked bool) {
	log.Printf("Adding clientOrderID %s to trade %s", orderId, trade.State.TradeID)
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

func (s *TradeService) UpdateSellableQuantity(trade *maker.Trade) {
	feeAsset := trade.FeeAsset()
	if feeAsset == "BNB" {
		trade.State.SellableQuantity = trade.State.BuyFillQuantity
	} else if feeAsset != "" {
		stepSize, err := s.binanceExchangeInfo.GetStepSize(trade.State.Symbol)
		if err != nil {
			log.WithError(err).WithField("symbol", trade.State.Symbol).
				Error("Failed to get symbol step size.")
		} else {
			trade.State.SellableQuantity = s.FixQuantityToStepSize(trade.State.BuyFillQuantity, stepSize)
		}
	}
}

func (s *TradeService) RestoreTrade(trade *maker.Trade) {
	s.TradesByLocalID[trade.State.TradeID] = trade
	for clientOrderId := range trade.State.ClientOrderIDs {
		s.TradesByClientID[clientOrderId] = trade
	}
	s.UpdateSellableQuantity(trade)
	s.binanceStreamManager.SubscribeTradeStream(trade.State.Symbol)
}

func (s *TradeService) AddNewTrade(trade *maker.Trade) (string) {
	if trade.State.TradeID == "" {
		localId, err := s.idGenerator.GetID(nil)
		if err != nil {
			log.Fatalf("error: failed to generate trade id: %v", err)
		}
		trade.State.TradeID = localId.String()
	}
	trade.State.Status = maker.TradeStatusNew

	s.lock.Lock()
	s.TradesByLocalID[trade.State.TradeID] = trade
	for clientOrderId := range trade.State.ClientOrderIDs {
		log.WithFields(log.Fields{
			"tradeId":       trade.State.TradeID,
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

	return trade.State.TradeID
}

func (s *TradeService) RemoveTrade(trade *maker.Trade) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.TradesByLocalID, trade.State.TradeID)
	for clientId := range trade.State.ClientOrderIDs {
		log.WithFields(log.Fields{
			"tradeId":       trade.State.TradeID,
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

// Note: Be sure to process reports even after a fill, as sometimes partial
//       fills will be received after the fill report.
func (s *TradeService) OnExecutionReport(event *UserStreamEvent) {
	report := event.ExecutionReport

	trade := s.FindTradeForReport(report)
	if trade == nil {
		log.Errorf("Failed to find trade for execution report: %s", log.ToJson(report))
		return
	}

	log.WithFields(log.Fields{
		"tradeId": trade.State.TradeID,
	}).Debugf("Received execution report: %s", log.ToJson(report))

	_, err := s.binanceExchangeInfo.GetStepSize(trade.State.Symbol)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"symbol": trade.State.Symbol,
		}).Error("Failed to get Binance symbol information.")
	}

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
			s.UpdateSellableQuantity(trade)
		case binance.OrderStatusFilled:
			trade.AddBuyFill(report)
			s.UpdateSellableQuantity(trade)
			trade.State.Status = maker.TradeStatusWatching
			if trade.State.LimitSell.Enabled {
				s.LimitSellByPercent(trade, trade.State.LimitSell.Percent)
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
	}

	db.DbUpdateTrade(trade)
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

// Return the sellable quantity adjusted for lot size.
func (s *TradeService) FixQuantityToStepSize(quantity float64, stepSize float64) float64 {
	fixedQuantity := roundx(quantity, 1/stepSize)
	if fixedQuantity > quantity {
		fixedQuantity = roundx(fixedQuantity-stepSize, 1/stepSize)
	}
	return fixedQuantity
}

func (s *TradeService) MarketSell(trade *maker.Trade, locked bool) error {
	quantity := trade.State.SellableQuantity - trade.State.SellFillQuantity

	clientOrderId, err := s.MakeOrderID()
	if err != nil {
		log.WithError(err).Errorf("Failed to generate order ID")
		return err
	}

	s.AddClientOrderId(trade, clientOrderId, locked)

	log.WithFields(log.Fields{
		"symbol":   trade.State.Symbol,
		"quantity": quantity,
		"tradeId":  trade.State.TradeID,
	}).Info("Posting market sell order.")

	order := binance.OrderParameters{
		Symbol:           trade.State.Symbol,
		Side:             binance.OrderSideSell,
		Type:             binance.OrderTypeMarket,
		Quantity:         quantity,
		NewClientOrderId: clientOrderId,
	}
	_, err = getBinanceRestClient().PostOrder(order)
	return err
}

func (s *TradeService) LimitSellByPercent(trade *maker.Trade, percent float64) error {
	symbolInfo, err := s.binanceExchangeInfo.GetSymbol(trade.State.Symbol)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"symbol": trade.State.Symbol,
		}).Error("Failed to get info for symbol.")
		return err
	}

	price := trade.State.BuyCost *
		(1 + trade.State.Fee) * (1 + (percent / 100)) /
		trade.State.SellableQuantity
	price = roundx(price, 1/symbolInfo.TickSize)

	tickSize := symbolInfo.TickSize
	if price <= trade.State.EffectiveBuyPrice {
		fixedPrice := price + tickSize
		log.WithFields(log.Fields{
			"tickSize":          tickSize,
			"symbol":            trade.State.Symbol,
			"price":             price,
			"effectiveBuyPrice": trade.State.EffectiveBuyPrice,
			"newPrice":          fixedPrice,
		}).Warnf("Sell price <= effective buy price, incrementing by tick size.")
		price = fixedPrice
	}

	clientOrderId, err := s.MakeOrderID()
	if err != nil {
		log.Printf("ERROR: Failed to generate clientOrderId: %v", err)
		return err
	}
	s.AddClientOrderId(trade, clientOrderId, false)

	quantity := trade.State.SellableQuantity - trade.State.SellFillQuantity

	log.WithFields(log.Fields{
		"price":    fmt.Sprintf("%.8f", price),
		"symbol":   trade.State.Symbol,
		"tradeId":  trade.State.TradeID,
		"quantity": quantity,
	}).Debugf("Posting limit sell order at percent.")

	order := binance.OrderParameters{
		Symbol:           trade.State.Symbol,
		Side:             binance.OrderSideSell,
		Type:             binance.OrderTypeLimit,
		TimeInForce:      binance.TimeInForceGTC,
		Quantity:         quantity,
		Price:            price,
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
		"price":           price,
		"symbol":          trade.State.Symbol,
		"tradeId":         trade.State.TradeID,
	}).Info("Sell order posted.")
	db.DbUpdateTrade(trade)
	return nil
}

func (s *TradeService) LimitSellByPrice(trade *maker.Trade, price float64) error {
	clientOrderId, err := s.MakeOrderID()
	if err != nil {
		log.Printf("ERROR: Failed to generate clientOrderId: %v", err)
		return err
	}
	s.AddClientOrderId(trade, clientOrderId, false)

	log.WithFields(log.Fields{
		"price":    fmt.Sprintf("%.8f", price),
		"symbol":   trade.State.Symbol,
		"tradeId":  trade.State.TradeID,
		"quantity": trade.State.SellableQuantity,
	}).Debugf("Posting limit sell order at price.")

	order := binance.OrderParameters{
		Symbol:           trade.State.Symbol,
		Side:             binance.OrderSideSell,
		Type:             binance.OrderTypeLimit,
		TimeInForce:      binance.TimeInForceGTC,
		Quantity:         trade.State.SellableQuantity,
		Price:            price,
		NewClientOrderId: clientOrderId,
	}
	_, err = getBinanceRestClient().PostOrder(order)
	if err != nil {
		log.WithFields(log.Fields{
		}).WithError(err).Error("Failed to send sell order.")
		return err
	}
	log.WithFields(log.Fields{
		"price":   price,
		"symbol":  trade.State.Symbol,
		"tradeId": trade.State.TradeID,
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

func (s *TradeService) UpdateTrailingProfit(trade *maker.Trade, enable bool,
	percent float64, deviation float64) {
	trade.SetTrailingProfit(enable, percent, deviation)
	db.DbUpdateTrade(trade)
	s.BroadcastTradeUpdate(trade)
}

func (s *TradeService) CancelSell(trade *maker.Trade) error {
	log.WithFields(log.Fields{
		"symbol":  trade.State.Symbol,
		"tradeId": trade.State.TradeID,
		"orderId": trade.State.SellOrderId,
	}).Info("Cancelling sell order.")
	log.Printf("Cancelling sell order: symbol=%s; orderId=%d",
		trade.State.Symbol, trade.State.SellOrderId)
	_, err := getBinanceRestClient().CancelOrder(
		trade.State.Symbol, trade.State.SellOrderId)
	return err
}

func roundx(val float64, x float64) float64 {
	return math.Round(val*x) / x
}
