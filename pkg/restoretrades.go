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
	"gitlab.com/crankykernel/maker/pkg/db"
	"gitlab.com/crankykernel/cryptotrader/binance"
	"gitlab.com/crankykernel/maker/pkg/maker"
	"gitlab.com/crankykernel/maker/pkg/log"
	"time"
)

func restoreTrades(tradeService *TradeService) {
	binanceRestClient := getBinanceRestClient()
	tradeStates, err := db.DbRestoreTradeState()
	if err != nil {
		log.Fatalf("error: failed to restore trade state: %v", err)
	}

	tradeHistoryCache := map[string][]binance.TradeResponse{}

	for _, state := range (tradeStates) {
		trade := maker.NewTradeWithState(state)
		tradeService.RestoreTrade(trade)

		if trade.State.Status == maker.TradeStatusPendingBuy {
			order, err := binanceRestClient.GetOrderByOrderId(
				trade.State.Symbol, trade.State.BuyOrderId)
			if err != nil {
				log.WithError(err).Error("Failed to get order by ID.")
			}
			switch order.Status {
			case binance.OrderStatusNew:
				// No change.
			default:
				log.WithFields(log.Fields{
					"tradeId":     trade.State.TradeID,
					"orderStatus": order.Status,
					"symbol":      trade.State.Symbol,
					"tradeStatus": trade.State.Status,
				}).Warnf("Don't know how to restore pending buy trade.")
			}
		}

		if trade.State.Status == maker.TradeStatusPendingSell {
			order, err := binanceRestClient.GetOrderByOrderId(
				trade.State.Symbol, trade.State.SellOrderId)
			if err != nil {
				log.WithError(err).Errorf(
					"Failed to find existing order %d for %s.",
					trade.State.SellOrderId, trade.State.Symbol)
			} else {
				if order.Status == binance.OrderStatusNew {
					// Unchanged.
				} else if order.Status == binance.OrderStatusCanceled {
					log.WithFields(log.Fields{
						"symbol":  state.Symbol,
						"tradeId": state.TradeID,
					}).Infof("Outstanding sell order has been canceled.")
					trade.State.Status = maker.TradeStatusWatching
				} else if order.Status == binance.OrderStatusFilled {
					trades := tradeHistoryCache[state.Symbol]
					if trades == nil {
						trades, err = binanceRestClient.GetMytrades(state.Symbol, 0, -1)
						if err != nil {
							log.Errorf("Failed to get trades: %v", err)
						}
						tradeHistoryCache[state.Symbol] = trades
					}
					for _, _trade := range trades {
						if _trade.OrderID == state.SellOrderId {
							fill := maker.OrderFill{
								Price:            _trade.Price,
								Quantity:         _trade.Quantity,
								CommissionAmount: _trade.Commission,
								CommissionAsset:  _trade.CommissionAsset,
							}
							trade.DoAddSellFill(fill)
						}
					}
					if trade.State.SellFillQuantity != trade.State.BuyFillQuantity {
						log.WithFields(log.Fields{
							"buyQuantity":  trade.State.BuyFillQuantity,
							"sellQuantity": trade.State.SellFillQuantity,
						}).Warnf("Order is filled but sell quantity != buy quantity.")
					} else {
						closeTime := time.Unix(0, order.TimeMillis*int64(time.Millisecond))
						log.WithFields(log.Fields{
							"symbol":    trade.State.Symbol,
							"closeTime": closeTime,
							"tradeId":   trade.State.TradeID,
						}).Infof("Closing trade.")
						tradeService.CloseTrade(trade, maker.TradeStatusDone, closeTime)
					}
				} else {
					log.WithFields(log.Fields{
						"symbol":      state.Symbol,
						"tradeId":     state.TradeID,
						"orderStatus": order.Status,
					}).Warnf("Don't know how to restore trade in status %v: %s",
						order.Status, log.ToJson(order))
				}
			}
		}
	}
	log.Printf("Restored %d trade states.", len(tradeService.TradesByClientID))
}
