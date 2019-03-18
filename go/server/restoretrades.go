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
	"github.com/crankykernel/binanceapi-go"
	"gitlab.com/crankykernel/maker/go/binanceex"
	"gitlab.com/crankykernel/maker/go/db"
	"gitlab.com/crankykernel/maker/go/log"
	"gitlab.com/crankykernel/maker/go/tradeservice"
	"gitlab.com/crankykernel/maker/go/types"
	"time"
)

func restoreTrades(tradeService *tradeservice.TradeService) {
	binanceRestClient := binanceex.GetBinanceRestClient()
	tradeStates, err := db.DbRestoreTradeState()
	if err != nil {
		log.Fatalf("error: failed to restore trade state: %v", err)
	}

	tradeHistoryCache := map[string][]binanceapi.MyTradesResponseEntry{}

	for _, state := range tradeStates {
		position := types.NewTradeWithState(state)
		tradeService.RestoreTrade(position)

		if position.State.Status == types.TradeStatusNew {
			var clientOrderId string = ""
			for clientOrderId = range position.State.ClientOrderIDs {
				break
			}
			order, err := binanceRestClient.GetOrderByClientId(position.State.Symbol, clientOrderId)
			if err != nil {
				log.WithError(err).Error("Failed to get trade by client order ID.")
				continue
			}

			switch order.Status {
			case binanceapi.OrderStatusNew:
				position.State.Status = types.TradeStatusPendingBuy
			case binanceapi.OrderStatusPartiallyFilled:
				position.State.Status = types.TradeStatusPendingBuy
				trades := tradeHistoryCache[state.Symbol]
				if trades == nil {
					trades, err = binanceRestClient.GetMytrades(state.Symbol, 0, -1)
					if err != nil {
						log.Errorf("Failed to get trades: %v", err)
					}
					tradeHistoryCache[state.Symbol] = trades
				}
				for _, trade := range trades {
					if trade.OrderID == order.OrderId {
						log.Println(log.ToJson(trade))
						fill := types.OrderFill{
							Price:            trade.Price,
							Quantity:         trade.Quantity,
							CommissionAsset:  trade.CommissionAsset,
							CommissionAmount: trade.Commission,
						}
						position.DoAddBuyFill(fill)
					}
				}
			case binanceapi.OrderStatusFilled:
				position.State.Status = types.TradeStatusWatching
				trades := tradeHistoryCache[state.Symbol]
				if trades == nil {
					trades, err = binanceRestClient.GetMytrades(state.Symbol, 0, -1)
					if err != nil {
						log.Errorf("Failed to get trades: %v", err)
					}
					tradeHistoryCache[state.Symbol] = trades
				}
				for _, trade := range trades {
					if trade.OrderID == order.OrderId {
						log.Println(log.ToJson(trade))
						fill := types.OrderFill{
							Price:            trade.Price,
							Quantity:         trade.Quantity,
							CommissionAsset:  trade.CommissionAsset,
							CommissionAmount: trade.Commission,
						}
						position.DoAddBuyFill(fill)
					}
				}
			default:
				log.Errorf("Don't know how to restore new trade now with status %s.",
					order.Status)
				log.Println(log.ToJson(order))
			}
		} else if position.State.Status == types.TradeStatusPendingBuy {
			order, err := binanceRestClient.GetOrderByOrderId(
				position.State.Symbol, position.State.BuyOrderId)
			if err != nil {
				log.WithError(err).Error("Failed to get order by ID.")
			}
			switch order.Status {
			case binanceapi.OrderStatusNew:
				// No change.
			default:
				log.WithFields(log.Fields{
					"tradeId":     position.State.TradeID,
					"orderStatus": order.Status,
					"symbol":      position.State.Symbol,
					"tradeStatus": position.State.Status,
				}).Warnf("Don't know how to restore pending buy trade.")
			}
		}

		if position.State.Status == types.TradeStatusPendingSell {
			order, err := binanceRestClient.GetOrderByOrderId(
				position.State.Symbol, position.State.SellOrderId)
			if err != nil {
				log.WithError(err).Errorf(
					"Failed to find existing order %d for %s.",
					position.State.SellOrderId, position.State.Symbol)
			} else {
				if order.Status == binanceapi.OrderStatusNew {
					// Unchanged.
				} else if order.Status == binanceapi.OrderStatusCanceled {
					log.WithFields(log.Fields{
						"symbol":  state.Symbol,
						"tradeId": state.TradeID,
					}).Infof("Outstanding sell order has been canceled.")
					position.State.Status = types.TradeStatusWatching
				} else if order.Status == binanceapi.OrderStatusFilled {
					trades := tradeHistoryCache[state.Symbol]
					if trades == nil {
						trades, err = binanceRestClient.GetMytrades(state.Symbol, 0, -1)
						if err != nil {
							log.WithError(err).Errorf("Failed to get trades from Binance")
						}
						tradeHistoryCache[state.Symbol] = trades
					}
					for _, trade := range trades {
						if trade.OrderID == state.SellOrderId {
							fill := types.OrderFill{
								Price:            trade.Price,
								Quantity:         trade.Quantity,
								CommissionAmount: trade.Commission,
								CommissionAsset:  trade.CommissionAsset,
							}
							position.DoAddSellFill(fill)
						}
					}
					if position.State.SellFillQuantity != position.State.BuyFillQuantity {
						log.WithFields(log.Fields{
							"buyQuantity":  position.State.BuyFillQuantity,
							"sellQuantity": position.State.SellFillQuantity,
						}).Warnf("Order is filled but sell quantity != buy quantity.")
					} else {
						closeTime := time.Unix(0, order.TimeMillis*int64(time.Millisecond))
						log.WithFields(log.Fields{
							"symbol":    position.State.Symbol,
							"closeTime": closeTime,
							"tradeId":   position.State.TradeID,
						}).Infof("Closing trade.")
						tradeService.CloseTrade(position, types.TradeStatusDone, closeTime)
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
		tradeService.UpdateSellableQuantity(position)
		db.DbUpdateTrade(position)
	}
	log.Printf("Restored %d trade states.", len(tradeService.TradesByClientID))
}
