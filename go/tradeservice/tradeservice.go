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

package tradeservice

import (
	"fmt"
	"gitlab.com/crankykernel/cryptotrader/binance"
	"gitlab.com/crankykernel/maker/binanceex"
	"gitlab.com/crankykernel/maker/db"
	"gitlab.com/crankykernel/maker/idgenerator"
	"gitlab.com/crankykernel/maker/log"
	"gitlab.com/crankykernel/maker/types"
	"gitlab.com/crankykernel/maker/util"
	"math"
	"sync"
	"time"
)

type TradeEventType string

const (
	TradeEventTypeUpdate  TradeEventType = "update"
	TradeEventTypeArchive TradeEventType = "archived"
)

type TradeEvent struct {
	EventType  TradeEventType
	TradeState *types.TradeState
	TradeID    string
}

type TradeService struct {
	// TradesByClientID by local ID. This map should only contain a single instance of
	// each trade.
	TradesByLocalID map[string]*types.Trade

	// TradesByClientID by client ID. This will contain multiple instances of the same
	// trade as a key is created for each client ID associated with the trade.
	TradesByClientID map[string]*types.Trade

	idGenerator *idgenerator.IdGenerator

	subscribers map[chan TradeEvent]bool
	lock        sync.RWMutex

	tradeStreamManager *binanceex.TradeStreamManager
	tradeStreamChannel binanceex.TradeStreamChannel

	binanceExchangeInfo *binance.ExchangeInfoService
}

func NewTradeService(binanceStreamManager *binanceex.TradeStreamManager) *TradeService {
	tradeService := &TradeService{
		TradesByLocalID:      make(map[string]*types.Trade),
		TradesByClientID:     make(map[string]*types.Trade),
		idGenerator:          idgenerator.NewIdGenerator(),
		subscribers:          make(map[chan TradeEvent]bool),
		tradeStreamManager: binanceStreamManager,
		binanceExchangeInfo:  binance.NewExchangeInfoService(),
	}

	if err := tradeService.binanceExchangeInfo.Update(); err != nil {
		log.Printf("error: failed to update exchange info: %v", err)
	}

	tradeService.tradeStreamChannel = tradeService.tradeStreamManager.Subscribe()

	go tradeService.tradeStreamListener()

	return tradeService
}

func (s *TradeService) tradeStreamListener() {
	for {
		select {
		case xlastTrade := <-s.tradeStreamChannel:
			s.onLastTrade(xlastTrade)
		}
	}
}

// Calculate the profit based on the trade being sold at the given price.
// Returns a percentage value in the range of 0-100.
func (s *TradeService) CalculateProfit(trade *types.Trade, price float64) float64 {
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
			case types.TradeStatusPendingSell:
			case types.TradeStatusWatching:
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

func (s *TradeService) checkStopLoss(trade *types.Trade) {
	switch trade.State.Status {
	case types.TradeStatusPendingSell:
	case types.TradeStatusWatching:
	default:
		return
	}
	if trade.State.StopLoss.Triggered {
		return
	}
	if trade.State.ProfitPercent < math.Abs(trade.State.StopLoss.Percent)*-1 {
		log.WithFields(log.Fields{
			"symbol": trade.State.Symbol,
			"loss":   trade.State.ProfitPercent,
		}).Infof("Stop Loss: Triggering market sell.")
		if trade.State.Status == types.TradeStatusPendingSell {
			s.CancelSell(trade)
		}
		trade.State.StopLoss.Triggered = true
		s.MarketSell(trade, true)
	}
}

func (s *TradeService) checkTrailingProfit(trade *types.Trade, price float64) {
	switch trade.State.Status {
	case types.TradeStatusPendingSell:
	case types.TradeStatusWatching:
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

func (s *TradeService) GetAllTrades() []*types.Trade {
	s.lock.RLock()
	defer s.lock.RUnlock()
	trades := []*types.Trade{}
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

func (s *TradeService) BroadcastTradeUpdate(trade *types.Trade) {
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

func (s *TradeService) FindTradeByLocalID(localId string) *types.Trade {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.TradesByLocalID[localId]
}

func (s *TradeService) AbandonTrade(trade *types.Trade) {
	if !trade.IsDone() {
		s.CloseTrade(trade, types.TradeStatusAbandoned, time.Now())
		s.BroadcastTradeUpdate(trade)
	}
}

func (s *TradeService) ArchiveTrade(trade *types.Trade) error {
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

func (s *TradeService) AddClientOrderId(trade *types.Trade, orderId string, locked bool) {
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

func (s *TradeService) UpdateSellableQuantity(trade *types.Trade) {
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

func (s *TradeService) RestoreTrade(trade *types.Trade) {
	s.TradesByLocalID[trade.State.TradeID] = trade
	for clientOrderId := range trade.State.ClientOrderIDs {
		s.TradesByClientID[clientOrderId] = trade
	}
	s.UpdateSellableQuantity(trade)
	s.tradeStreamManager.AddSymbol(trade.State.Symbol)
}

func (s *TradeService) AddNewTrade(trade *types.Trade) (string) {
	if trade.State.TradeID == "" {
		localId, err := s.idGenerator.GetID(nil)
		if err != nil {
			log.Fatalf("error: failed to generate trade id: %v", err)
		}
		trade.State.TradeID = localId.String()
	}
	trade.State.Status = types.TradeStatusNew

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

	s.tradeStreamManager.AddSymbol(trade.State.Symbol)
	s.BroadcastTradeUpdate(trade)

	lastPrice, err := binance.NewAnonymousClient().GetPriceTicker(trade.State.Symbol)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"symbol": trade.State.Symbol,
		}).Errorf("Failed to get last price for new trade")
	} else {
		if trade.State.LastPrice == 0 {
			trade.State.LastPrice = lastPrice.Price
			s.BroadcastTradeUpdate(trade)
		}
	}

	return trade.State.TradeID
}

func (s *TradeService) RemoveTrade(trade *types.Trade) {
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

func (s *TradeService) FailTrade(trade *types.Trade) {
	s.CloseTrade(trade, types.TradeStatusFailed, time.Now())
	s.BroadcastTradeUpdate(trade)
}

func (s *TradeService) FindTradeForReport(report binance.StreamExecutionReport) *types.Trade {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if trade, ok := s.TradesByClientID[report.ClientOrderID]; ok {
		return trade
	}

	if trade, ok := s.TradesByClientID[report.OriginalClientOrderID]; ok {
		return trade
	}

	log.WithFields(log.Fields{
		"clientOrderId":     report.ClientOrderID,
		"origClientOrderId": report.OriginalClientOrderID,
	}).Debugf("Failed to find trade by orderId")

	return nil
}

// Note: Be sure to process reports even after a fill, as sometimes partial
//       fills will be received after the fill report.
func (s *TradeService) OnExecutionReport(event *binanceex.UserStreamEvent) {
	report := event.ExecutionReport

	trade := s.FindTradeForReport(report)
	if trade == nil {
		log.Errorf("Failed to find trade for execution report: %s", log.ToJson(report))
		return
	}

	log.WithFields(log.Fields{
		"tradeId": trade.State.TradeID,
		"symbol":  trade.State.Symbol,
	}).Debugf("Received execution report: %s", log.ToJson(report))

	trade.AddHistory(types.HistoryEntry{
		Timestamp: time.Now(),
		Type:      types.HistoryTypeExecutionReport,
		Fields:    report,
	})

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
			if trade.State.Status == types.TradeStatusNew {
				trade.State.Status = types.TradeStatusPendingBuy
			}
			if trade.State.LastBuyStatus == "" {
				trade.State.LastBuyStatus = report.CurrentOrderStatus
			}
		case binance.OrderStatusCanceled:
			if trade.State.BuyFillQuantity == 0 {
				trade.State.Status = types.TradeStatusCanceled
			} else {
				trade.State.Status = types.TradeStatusWatching
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
			trade.State.Status = types.TradeStatusWatching
			trade.State.LastBuyStatus = report.CurrentOrderStatus
			s.TriggerLimitSell(trade)
		}

	case binance.OrderSideSell:
		switch report.CurrentOrderStatus {
		case binance.OrderStatusNew:
			if trade.State.Status == types.TradeStatusDone {
				// Sometimes we get the fill before the new.
				break
			}
			trade.State.SellOrderId = report.OrderID
			trade.State.Status = types.TradeStatusPendingSell
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
			fill := types.OrderFill{
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
			fill := types.OrderFill{
				Price:            report.LastExecutedPrice,
				Quantity:         report.LastExecutedQuantity,
				CommissionAsset:  report.CommissionAsset,
				CommissionAmount: report.CommissionAmount,
			}
			trade.DoAddSellFill(fill)
			trade.State.Status = types.TradeStatusDone
			trade.State.SellOrder.Status = report.CurrentOrderStatus
		case binance.OrderStatusCanceled:
			trade.State.Status = types.TradeStatusWatching
			trade.State.SellOrder.Status = report.CurrentOrderStatus
		default:
			log.Printf("ERROR: Unhandled order status state %s for sell side.",
				report.CurrentOrderStatus)
			trade.State.SellOrder.Status = report.CurrentOrderStatus
		}
	}

	switch trade.State.Status {
	case types.TradeStatusDone:
		fallthrough
	case types.TradeStatusCanceled:
		fallthrough
	case types.TradeStatusFailed:
		trade.State.CloseTime = &event.EventTime
		s.tradeStreamManager.RemoveSymbol(trade.State.Symbol)
	}

	db.DbUpdateTrade(trade)
	s.BroadcastTradeUpdate(trade)
}

func (s *TradeService) TriggerLimitSell(trade *types.Trade) {
	if trade.State.LimitSell.Enabled {
		if trade.State.LimitSell.Type == types.LimitSellTypePercent {
			log.WithFields(log.Fields{
				"tradeId": trade.State.TradeID,
				"symbol":  trade.State.Symbol,
			}).Infof("Triggering limit sell at %f percent.",
				trade.State.LimitSell.Percent)
			s.LimitSellByPercent(trade, trade.State.LimitSell.Percent)
		} else if trade.State.LimitSell.Type == types.LimitSellTypePrice {
			log.WithFields(log.Fields{
				"tradeId": trade.State.TradeID,
				"symbol":  trade.State.Symbol,
			}).Infof("Triggering limit sell at price %f.",
				trade.State.LimitSell.Price)
			s.LimitSellByPrice(trade, trade.State.LimitSell.Price)
		} else {
			log.WithFields(log.Fields{
				"tradeId": trade.State.TradeID,
				"symbol":  trade.State.Symbol,
			}).Errorf("Unknown limit sell type: %v", trade.State.LimitSell.Type)
		}
	} else {
		log.WithFields(log.Fields{
			"tradeId": trade.State.TradeID,
			"symbol":  trade.State.Symbol,
		}).Debug("Limit sell not enabled.")
	}
}

func (s *TradeService) CloseTrade(trade *types.Trade, status types.TradeStatus, closeTime time.Time) {
	if closeTime.IsZero() {
		closeTime = time.Now()
	}
	trade.State.Status = status
	trade.State.CloseTime = &closeTime
	s.tradeStreamManager.RemoveSymbol(trade.State.Symbol)
	db.DbUpdateTrade(trade)
}

// Return the sellable quantity adjusted for lot size.
func (s *TradeService) FixQuantityToStepSize(quantity float64, stepSize float64) float64 {
	fixedQuantity := util.Roundx(quantity, 1/stepSize)
	if fixedQuantity > quantity {
		fixedQuantity = util.Roundx(fixedQuantity-stepSize, 1/stepSize)
	}
	return fixedQuantity
}

func (s *TradeService) MarketSell(trade *types.Trade, locked bool) error {
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
	_, err = binanceex.GetBinanceRestClient().PostOrder(order)
	return err
}

func (s *TradeService) LimitSellByPercent(trade *types.Trade, percent float64) error {
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
	price = util.Roundx(price, 1/symbolInfo.TickSize)

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
	_, err = binanceex.GetBinanceRestClient().PostOrder(order)
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

	trade.AddHistoryEntry(types.HistoryTypeSellOrder, map[string]interface{}{
		"sellOrderType": "limitSellByPercent",
		"percent":       percent,
		"price":         price,
		"quantity":      quantity,
		"clientOrderId": clientOrderId,
	})

	trade.State.LimitSell.Enabled = true
	trade.State.LimitSell.Percent = percent
	trade.State.LimitSell.Price = price
	trade.State.LimitSell.Type = types.LimitSellTypePercent

	db.DbUpdateTrade(trade)
	s.BroadcastTradeUpdate(trade)

	return nil
}

func (s *TradeService) LimitSellByPrice(trade *types.Trade, price float64) error {
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
	_, err = binanceex.GetBinanceRestClient().PostOrder(order)
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

func (s *TradeService) UpdateStopLoss(trade *types.Trade, enable bool, percent float64) {
	trade.SetStopLoss(enable, percent)
	log.WithFields(log.Fields{
		"symbol":  trade.State.Symbol,
		"tradeId": trade.State.TradeID,
		"enable":  enable,
		"percent": percent,
	}).Infof("Stop loss settings updated")
	trade.AddHistoryEntry(types.HistoryTypeStopLossUpdate, map[string]interface{}{
		"enable":  enable,
		"percent": percent,
	})
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

func (s *TradeService) UpdateTrailingProfit(trade *types.Trade, enable bool,
	percent float64, deviation float64) {
	trade.SetTrailingProfit(enable, percent, deviation)
	log.WithFields(log.Fields{
		"symbol":    trade.State.Symbol,
		"tradeId":   trade.State.TradeID,
		"percent":   percent,
		"deviation": deviation,
		"enabled":   enable,
	}).Infof("Trailing profit settings updated")
	trade.AddHistoryEntry(types.HistoryTypeTrailingProfitUpdate, map[string]interface{}{
		"enable":    enable,
		"percent":   percent,
		"deviation": deviation,
	})
	db.DbUpdateTrade(trade)
	s.BroadcastTradeUpdate(trade)
}

func (s *TradeService) CancelSell(trade *types.Trade) error {
	log.WithFields(log.Fields{
		"symbol":  trade.State.Symbol,
		"tradeId": trade.State.TradeID,
		"orderId": trade.State.SellOrderId,
	}).Info("Cancelling sell order.")
	log.Printf("Cancelling sell order: symbol=%s; orderId=%d",
		trade.State.Symbol, trade.State.SellOrderId)
	_, err := binanceex.GetBinanceRestClient().CancelOrder(
		trade.State.Symbol, trade.State.SellOrderId)
	if err == nil {
		trade.AddHistoryEntry(types.HistoryTypeSellCanceled, map[string]interface{}{
			"sellOrderId": trade.State.SellOrderId,
			"success":     true,
		})
		trade.State.LimitSell.Enabled = false
	} else {
		log.WithError(err).WithFields(log.Fields{
			"symbol":  trade.State.Symbol,
			"tradeId": trade.State.TradeID,
			"orderId": trade.State.SellOrderId,
		}).Errorf("Failed to cancel sell order")
		trade.AddHistoryEntry(types.HistoryTypeSellCanceled, map[string]interface{}{
			"sellOrderId": trade.State.SellOrderId,
			"success":     false,
			"error":       fmt.Sprintf("%v", err),
		})
	}
	db.DbUpdateTrade(trade)
	s.BroadcastTradeUpdate(trade)
	return err
}

func (s *TradeService) CancelBuy(trade *types.Trade) error {
	_, err := binanceex.GetBinanceRestClient().CancelOrder(
		trade.State.Symbol, trade.State.BuyOrderId)
	if err != nil {
		trade.AddHistoryEntry(types.HistoryTypeBuyCanceled, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("%v", err),
		})
	} else {
		trade.AddHistoryEntry(types.HistoryTypeBuyCanceled, map[string]interface{}{
			"success": true,
		})
	}
	db.DbUpdateTrade(trade)
	return err
}
