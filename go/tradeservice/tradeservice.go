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
	"github.com/crankykernel/binanceapi-go"
	"gitlab.com/crankykernel/maker/go/binanceex"
	"gitlab.com/crankykernel/maker/go/db"
	"gitlab.com/crankykernel/maker/go/idgenerator"
	"gitlab.com/crankykernel/maker/go/log"
	"gitlab.com/crankykernel/maker/go/types"
	"gitlab.com/crankykernel/maker/go/util"
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
	TradeState types.TradeState
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
	lock        sync.Mutex

	tradeStreamManager *binanceex.TradeStreamManager
	tradeStreamChannel binanceex.TradeStreamChannel

	binanceExchangeInfo *binanceex.ExchangeInfoService
}

func NewTradeService(binanceStreamManager *binanceex.TradeStreamManager) *TradeService {
	tradeService := &TradeService{
		TradesByLocalID:     make(map[string]*types.Trade),
		TradesByClientID:    make(map[string]*types.Trade),
		idGenerator:         idgenerator.NewIdGenerator(),
		subscribers:         make(map[chan TradeEvent]bool),
		tradeStreamManager:  binanceStreamManager,
		binanceExchangeInfo: binanceex.NewExchangeInfoService(),
	}

	if err := tradeService.binanceExchangeInfo.Update(); err != nil {
		log.WithError(err).
			Errorf("Failed to update Binance exchange info")
	}

	tradeService.tradeStreamChannel = tradeService.tradeStreamManager.Subscribe()

	go tradeService.tradeStreamListener()

	return tradeService
}

// Calculate the profit based on the trade being sold at the given price.
// Returns a percentage value in the range of 0-100.
func (s *TradeService) CalculateProfit(trade *types.Trade, price float64) float64 {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.calculateProfit(trade, price)
}

func (s *TradeService) calculateProfit(trade *types.Trade, price float64) float64 {
	grossSellCost := price * trade.State.SellableQuantity
	netSellCost := grossSellCost * (1 - trade.State.Fee)
	profit := (netSellCost - trade.State.BuyCost) / trade.State.BuyCost * 100
	return profit
}

func (s *TradeService) tradeStreamListener() {
	for {
		lastTrade := <-s.tradeStreamChannel
		// Events come externally, must lock.
		s.lock.Lock()
		s.onLastTrade(lastTrade)
		s.lock.Unlock()
	}
}

func (s *TradeService) onLastTrade(lastTrade binanceapi.StreamAggTrade) {
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
			trade.State.ProfitPercent = s.calculateProfit(trade, lastTrade.Price)

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
			s.cancelSell(trade)
		}
		trade.State.StopLoss.Triggered = true
		s.marketSell(trade)
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

			log.WithFields(log.Fields{
				"symbol":    trade.State.Symbol,
				"deviation": deviation,
				"allowed":   trade.State.TrailingProfit.Deviation,
			}).Infof("Trailing profit update")

			if math.Abs(deviation) > trade.State.TrailingProfit.Deviation {
				log.WithFields(log.Fields{
					"symbol":  trade.State.Symbol,
					"percent": trade.State.ProfitPercent,
				}).Infof("Executing trailing profit sell")
				trade.State.TrailingProfit.Triggered = true
				s.marketSell(trade)
			}
		}
	} else {
		if trade.State.ProfitPercent > trade.State.TrailingProfit.Percent {
			log.WithFields(log.Fields{
				"symbol":                  trade.State.Symbol,
				"currentProfit":           trade.State.ProfitPercent,
				"trailingProfitPercent":   trade.State.TrailingProfit.Percent,
				"trailingProfitDeviation": trade.State.TrailingProfit.Deviation,
			}).Infof("Activating trailing profit")
			trade.State.TrailingProfit.Activated = true
			s.broadcastTradeUpdate(trade)
		}
	}

}

func (s *TradeService) GetAllTrades() []*types.Trade {
	s.lock.Lock()
	defer s.lock.Unlock()
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
	s.subscribers[channel] = false
	delete(s.subscribers, channel)
}

func (s *TradeService) broadcastTradeEvent(tradeEvent TradeEvent) {
	for channel := range s.subscribers {
		channel <- tradeEvent
	}
}

func (s *TradeService) BroadcastTradeUpdate(trade *types.Trade) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.broadcastTradeUpdate(trade)
}

func (s *TradeService) broadcastTradeUpdate(trade *types.Trade) {
	tradeEvent := TradeEvent{
		EventType:  TradeEventTypeUpdate,
		TradeState: trade.State,
	}
	s.broadcastTradeEvent(tradeEvent)
}

func (s *TradeService) BroadcastTradeArchived(tradeId string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.broadcastTradeArchived(tradeId)
}

func (s *TradeService) broadcastTradeArchived(tradeId string) {
	tradeEvent := TradeEvent{
		EventType: TradeEventTypeArchive,
		TradeID:   tradeId,
	}
	s.broadcastTradeEvent(tradeEvent)
}

func (s *TradeService) FindTradeByLocalID(localId string) *types.Trade {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.findTradeByLocalID(localId)
}

func (s *TradeService) findTradeByLocalID(localId string) *types.Trade {
	return s.TradesByLocalID[localId]
}

func (s *TradeService) AbandonTrade(trade *types.Trade) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.abandonTrade(trade)
}

func (s *TradeService) abandonTrade(trade *types.Trade) {
	if !trade.IsDone() {
		s.closeTrade(trade, types.TradeStatusAbandoned, time.Now())
		s.BroadcastTradeUpdate(trade)
	}
}

func (s *TradeService) ArchiveTrade(trade *types.Trade) error {
	if trade.IsDone() {
		if err := db.DbArchiveTrade(trade); err != nil {
			return err
		}
		s.removeTrade(trade)
		s.BroadcastTradeArchived(trade.State.TradeID)
		return nil
	}
	return fmt.Errorf("archive not allowed in state %s", trade.State.Status)
}

func (s *TradeService) AddClientOrderId(trade *types.Trade, orderId string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.addClientOrderId(trade, orderId)
}

func (s *TradeService) addClientOrderId(trade *types.Trade, orderId string) {
	log.WithFields(log.Fields{
		"symbol":  trade.State.Symbol,
		"orderId": trade.State.TradeID,
	}).Debugf("Adding client order ID to trade")
	trade.AddClientOrderID(orderId)
	s.TradesByClientID[orderId] = trade
	db.DbUpdateTrade(trade)
}

func (s *TradeService) UpdateSellableQuantity(trade *types.Trade) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.updateSellableQuantity(trade)
}

func (s *TradeService) updateSellableQuantity(trade *types.Trade) {
	feeAsset := trade.FeeAsset()
	if feeAsset == "BNB" {
		trade.State.SellableQuantity = trade.State.BuyFillQuantity
	} else if feeAsset != "" {
		stepSize, err := s.binanceExchangeInfo.GetStepSize(trade.State.Symbol)
		if err != nil {
			log.WithError(err).WithField("symbol", trade.State.Symbol).
				Error("Failed to get symbol step size.")
		} else {
			trade.State.SellableQuantity = fixQuantityToStepSize(trade.State.BuyFillQuantity, stepSize)
		}
	}
}

func (s *TradeService) RestoreTrade(trade *types.Trade) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.TradesByLocalID[trade.State.TradeID] = trade
	for clientOrderId := range trade.State.ClientOrderIDs {
		s.TradesByClientID[clientOrderId] = trade
	}
	s.updateSellableQuantity(trade)
	if !trade.IsDone() {
		s.tradeStreamManager.AddSymbol(trade.State.Symbol)
	}
}

func (s *TradeService) AddNewTrade(trade *types.Trade) string {
	s.lock.Lock()
	defer s.lock.Unlock()
	if trade.State.TradeID == "" {
		localId, err := s.idGenerator.GetID(nil)
		if err != nil {
			log.Fatalf("error: failed to generate trade id: %v", err)
		}
		trade.State.TradeID = localId.String()
	}
	trade.State.Status = types.TradeStatusNew

	s.TradesByLocalID[trade.State.TradeID] = trade
	for clientOrderId := range trade.State.ClientOrderIDs {
		log.WithFields(log.Fields{
			"tradeId":       trade.State.TradeID,
			"clientOrderId": clientOrderId,
		}).Debugf("Recording clientOrderId for new trade.")
		s.TradesByClientID[clientOrderId] = trade
	}

	if err := db.DbSaveTrade(trade); err != nil {
		log.WithError(err).Errorf("Failed to save trade to database")
	}

	s.tradeStreamManager.AddSymbol(trade.State.Symbol)
	s.broadcastTradeUpdate(trade)

	lastPrice, err := binanceapi.NewRestClient().GetPriceTicker(trade.State.Symbol)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"symbol": trade.State.Symbol,
		}).Errorf("Failed to get last price for new trade")
	} else {
		if trade.State.LastPrice == 0 {
			trade.State.LastPrice = lastPrice.Price
			s.broadcastTradeUpdate(trade)
		}
	}

	return trade.State.TradeID
}

func (s *TradeService) RemoveTrade(trade *types.Trade) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.removeTrade(trade)
}

func (s *TradeService) removeTrade(trade *types.Trade) {
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
	s.lock.Lock()
	defer s.lock.Unlock()
	s.closeTrade(trade, types.TradeStatusFailed, time.Now())
	s.broadcastTradeUpdate(trade)
}

func (s *TradeService) FindTradeForReport(report binanceapi.StreamExecutionReport) *types.Trade {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.findTradeForReport(report)
}

func (s *TradeService) findTradeForReport(report binanceapi.StreamExecutionReport) *types.Trade {
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
	s.lock.Lock()
	defer s.lock.Unlock()

	report := event.ExecutionReport

	trade := s.findTradeForReport(report)
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
	case binanceapi.OrderSideBuy:
		switch report.CurrentOrderStatus {
		case binanceapi.OrderStatusNew:
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
		case binanceapi.OrderStatusCanceled:
			if trade.State.BuyFillQuantity == 0 {
				trade.State.Status = types.TradeStatusCanceled
			} else {
				trade.State.Status = types.TradeStatusWatching
			}
			trade.State.LastBuyStatus = report.CurrentOrderStatus
		case binanceapi.OrderStatusPartiallyFilled:
			if trade.State.LastBuyStatus != binanceapi.OrderStatusFilled {
				trade.State.LastBuyStatus = report.CurrentOrderStatus
			}
			trade.AddBuyFill(report)
			s.updateSellableQuantity(trade)
		case binanceapi.OrderStatusFilled:
			trade.AddBuyFill(report)
			s.updateSellableQuantity(trade)
			trade.State.Status = types.TradeStatusWatching
			trade.State.LastBuyStatus = report.CurrentOrderStatus
			s.triggerLimitSell(trade)
		}

	case binanceapi.OrderSideSell:
		switch report.CurrentOrderStatus {
		case binanceapi.OrderStatusNew:
			if trade.State.Status == types.TradeStatusDone {
				// Sometimes we get the fill before the new.
				break
			}
			trade.State.SellOrderId = report.OrderID
			trade.State.Status = types.TradeStatusPendingSell
			switch trade.State.SellOrder.Status {
			case binanceapi.OrderStatusPartiallyFilled:
			case binanceapi.OrderStatusFilled:
			default:
				trade.State.SellOrder.Status = report.CurrentOrderStatus
			}
			trade.State.SellOrder.Type = report.OrderType
			trade.State.SellOrder.Quantity = report.Quantity
			trade.State.SellOrder.Price = report.Price
		case binanceapi.OrderStatusPartiallyFilled:
			fill := types.OrderFill{
				Price:            report.LastExecutedPrice,
				Quantity:         report.LastExecutedQuantity,
				CommissionAsset:  report.CommissionAsset,
				CommissionAmount: report.CommissionAmount,
			}
			trade.DoAddSellFill(fill)
			if trade.State.SellOrder.Status != binanceapi.OrderStatusFilled {
				trade.State.SellOrder.Status = report.CurrentOrderStatus
			}
		case binanceapi.OrderStatusFilled:
			fill := types.OrderFill{
				Price:            report.LastExecutedPrice,
				Quantity:         report.LastExecutedQuantity,
				CommissionAsset:  report.CommissionAsset,
				CommissionAmount: report.CommissionAmount,
			}
			trade.DoAddSellFill(fill)
			trade.State.Status = types.TradeStatusDone
			trade.State.SellOrder.Status = report.CurrentOrderStatus
		case binanceapi.OrderStatusCanceled:
			trade.State.Status = types.TradeStatusWatching
			trade.State.SellOrder.Status = report.CurrentOrderStatus
		default:
			log.WithFields(log.Fields{
				"symbol":             trade.State.Symbol,
				"currentOrderStatus": report.CurrentOrderStatus,
				"side":               "sell",
			}).Errorf("Unknown current order status in execution report")
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
	s.broadcastTradeUpdate(trade)
}

func (s *TradeService) triggerLimitSell(trade *types.Trade) {
	if trade.State.LimitSell.Enabled {
		if trade.State.LimitSell.Type == types.LimitSellTypePercent {
			log.WithFields(log.Fields{
				"tradeId": trade.State.TradeID,
				"symbol":  trade.State.Symbol,
			}).Infof("Triggering limit sell at %f percent.",
				trade.State.LimitSell.Percent)
			s.limitSellByPercent(trade, trade.State.LimitSell.Percent)
		} else if trade.State.LimitSell.Type == types.LimitSellTypePrice {
			log.WithFields(log.Fields{
				"tradeId": trade.State.TradeID,
				"symbol":  trade.State.Symbol,
			}).Infof("Triggering limit sell at price %f.",
				trade.State.LimitSell.Price)
			s.limitSellByPrice(trade, trade.State.LimitSell.Price)
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
	s.lock.Lock()
	defer s.lock.Unlock()
	s.closeTrade(trade, status, closeTime)
}

func (s *TradeService) closeTrade(trade *types.Trade, status types.TradeStatus, closeTime time.Time) {
	if closeTime.IsZero() {
		closeTime = time.Now()
	}
	trade.State.Status = status
	trade.State.CloseTime = &closeTime
	s.tradeStreamManager.RemoveSymbol(trade.State.Symbol)
	db.DbUpdateTrade(trade)
}

func (s *TradeService) MarketSell(trade *types.Trade, locked bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.marketSell(trade)
}

func (s *TradeService) marketSell(trade *types.Trade) error {
	quantity := trade.State.SellableQuantity - trade.State.SellFillQuantity

	clientOrderId, err := s.MakeOrderID()
	if err != nil {
		log.WithError(err).Errorf("Failed to generate order ID")
		return err
	}

	s.addClientOrderId(trade, clientOrderId)

	log.WithFields(log.Fields{
		"symbol":   trade.State.Symbol,
		"quantity": quantity,
		"tradeId":  trade.State.TradeID,
	}).Info("Posting market sell order.")

	order := binanceapi.OrderParameters{
		Symbol:           trade.State.Symbol,
		Side:             binanceapi.OrderSideSell,
		Type:             binanceapi.OrderTypeMarket,
		Quantity:         quantity,
		NewClientOrderId: clientOrderId,
	}
	_, err = binanceex.GetBinanceRestClient().PostOrder(order)
	return err
}

func (s *TradeService) LimitSellByPercent(trade *types.Trade, percent float64) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.limitSellByPercent(trade, percent)
}

func (s *TradeService) limitSellByPercent(trade *types.Trade, percent float64) error {
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
		log.WithError(err).Errorf("Failed to generate clientOrderId")
		return err
	}
	s.addClientOrderId(trade, clientOrderId)

	quantity := trade.State.SellableQuantity - trade.State.SellFillQuantity

	log.WithFields(log.Fields{
		"price":    fmt.Sprintf("%.8f", price),
		"symbol":   trade.State.Symbol,
		"tradeId":  trade.State.TradeID,
		"quantity": quantity,
	}).Debugf("Posting limit sell order at percent.")

	order := binanceapi.OrderParameters{
		Symbol:           trade.State.Symbol,
		Side:             binanceapi.OrderSideSell,
		Type:             binanceapi.OrderTypeLimit,
		TimeInForce:      binanceapi.TimeInForceGTC,
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
	s.broadcastTradeUpdate(trade)

	return nil
}

func (s *TradeService) LimitSellByPrice(trade *types.Trade, price float64) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.limitSellByPrice(trade, price)
}

func (s *TradeService) limitSellByPrice(trade *types.Trade, price float64) error {
	clientOrderId, err := s.MakeOrderID()
	if err != nil {
		log.WithError(err).Errorf("Failed to generate clientOrderId")
		return err
	}
	s.addClientOrderId(trade, clientOrderId)

	log.WithFields(log.Fields{
		"price":    fmt.Sprintf("%.8f", price),
		"symbol":   trade.State.Symbol,
		"tradeId":  trade.State.TradeID,
		"quantity": trade.State.SellableQuantity,
	}).Debugf("Posting limit sell order at price.")

	order := binanceapi.OrderParameters{
		Symbol:           trade.State.Symbol,
		Side:             binanceapi.OrderSideSell,
		Type:             binanceapi.OrderTypeLimit,
		TimeInForce:      binanceapi.TimeInForceGTC,
		Quantity:         trade.State.SellableQuantity,
		Price:            price,
		NewClientOrderId: clientOrderId,
	}
	_, err = binanceex.GetBinanceRestClient().PostOrder(order)
	if err != nil {
		log.WithFields(log.Fields{}).WithError(err).Error("Failed to send sell order.")
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
	s.lock.Lock()
	defer s.lock.Unlock()
	s.updateStopLoss(trade, enable, percent)
	s.broadcastTradeUpdate(trade)
}

func (s *TradeService) updateStopLoss(trade *types.Trade, enable bool, percent float64) {
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
	s.broadcastTradeUpdate(trade)
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
	s.lock.Lock()
	defer s.lock.Unlock()
	s.updateTrailingProfit(trade, enable, percent, deviation)
}

func (s *TradeService) updateTrailingProfit(trade *types.Trade, enable bool,
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
	s.broadcastTradeUpdate(trade)
}

func (s *TradeService) CancelSell(trade *types.Trade) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.cancelSell(trade)
}

func (s *TradeService) cancelSell(trade *types.Trade) error {
	log.WithFields(log.Fields{
		"symbol":  trade.State.Symbol,
		"tradeId": trade.State.TradeID,
		"orderId": trade.State.SellOrderId,
	}).Info("Cancelling sell order.")
	_, err := binanceex.GetBinanceRestClient().CancelOrderById(
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
	s.broadcastTradeUpdate(trade)
	return err
}

func (s *TradeService) CancelBuy(trade *types.Trade) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.cancelBuy(trade)
}

func (s *TradeService) cancelBuy(trade *types.Trade) error {
	_, err := binanceex.GetBinanceRestClient().CancelOrderById(
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

// Return the sellable quantity adjusted for lot size.
func fixQuantityToStepSize(quantity float64, stepSize float64) float64 {
	fixedQuantity := util.Roundx(quantity, 1/stepSize)
	if fixedQuantity > quantity {
		fixedQuantity = util.Roundx(fixedQuantity-stepSize, 1/stepSize)
	}
	return fixedQuantity
}
