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

package types

import (
	"time"
	"math"
	"gitlab.com/crankykernel/cryptotrader/binance"
)

const TRADE_STATE_VERSION = 1

const DEFAULT_FEE = float64(0.001)
const BNB_FEE = float64(0.00075)

type Trade struct {
	State TradeState
}

func NewTrade() *Trade {
	return &Trade{
		State: TradeState{
			Version:        TRADE_STATE_VERSION,
			Status:         TradeStatusNew,
			Fee:            DEFAULT_FEE,
			OpenTime:       time.Now(),
			ClientOrderIDs: make(map[string]bool),
		},
	}
}

func NewTradeWithState(state TradeState) *Trade {
	if state.ClientOrderIDs == nil {
		state.ClientOrderIDs = make(map[string]bool)
	}
	trade := &Trade{
		State: state,
	}
	trade.UpdateSellState()
	trade.UpdateBuyState()
	return trade
}

func (t *Trade) IsDone() bool {
	switch (t.State.Status) {
	case TradeStatusDone:
	case TradeStatusCanceled:
	case TradeStatusFailed:
	case TradeStatusAbandoned:
	default:
		return false
	}
	return true
}

func (s *Trade) FeeAsset() string {
	lastFillIndex := len(s.State.BuySideFills) - 1
	if lastFillIndex < 0 {
		return ""
	}
	return s.State.BuySideFills[lastFillIndex].CommissionAsset
}

func (t *Trade) SetLimitSellByPercent(percent float64) {
	t.State.LimitSell.Enabled = true
	t.State.LimitSell.Type = LimitSellTypePercent
	t.State.LimitSell.Percent = percent
}

func (t *Trade) SetLimitSellByPrice(price float64) {
	t.State.LimitSell.Enabled = true
	t.State.LimitSell.Type = LimitSellTypePrice
	t.State.LimitSell.Price = price
}

func (t *Trade) SetStopLoss(enable bool, percent float64) {
	t.State.StopLoss.Enabled = enable
	t.State.StopLoss.Percent = percent
}

func (t *Trade) SetTrailingProfit(enable bool, percent float64, deviation float64) {
	t.State.TrailingProfit.Enabled = enable
	t.State.TrailingProfit.Percent = percent
	t.State.TrailingProfit.Deviation = deviation
}

func (t *Trade) AddBuyFill(report binance.StreamExecutionReport) {
	fill := OrderFill{
		Price:            report.LastExecutedPrice,
		Quantity:         report.LastExecutedQuantity,
		CommissionAmount: report.CommissionAmount,
		CommissionAsset:  report.CommissionAsset,
	}
	t.DoAddBuyFill(fill)
}

func (t *Trade) DoAddBuyFill(fill OrderFill) {
	t.State.BuySideFills = append(t.State.BuySideFills, fill)
	t.UpdateBuyState()
}

func (t *Trade) DoAddSellFill(fill OrderFill) {
	t.State.SellSideFills = append(t.State.SellSideFills, fill)
	t.UpdateSellState()
}

func (t *Trade) AddClientOrderID(clientOrderID string) {
	t.State.ClientOrderIDs[clientOrderID] = true
}

func (t *Trade) UpdateSellState() {
	quantity := float64(0)
	totalPrice := float64(0)
	cost := float64(0)

	for _, fill := range t.State.SellSideFills {
		quantity += fill.Quantity
		totalPrice += fill.Price * fill.Quantity
		if fill.CommissionAsset == "BNB" {
			cost += (fill.Price * fill.Quantity) * (1 - BNB_FEE)
		} else {
			cost += (fill.Price * fill.Quantity) - fill.CommissionAmount
		}
	}

	if quantity > 0 {
		t.State.AverageSellPrice = round8(totalPrice / quantity)
		t.State.SellFillQuantity = round8(quantity)
		t.State.SellCost = round8(cost)
		t.State.Profit = t.State.SellCost - t.State.BuyCost
		t.State.ProfitPercent = t.State.Profit / t.State.BuyCost * 100
		t.State.Profit = round8(t.State.Profit)
		t.State.ProfitPercent = round8(t.State.ProfitPercent)
	}
}

func (t *Trade) UpdateBuyState() {
	var cost float64 = 0
	var totalPrice float64 = 0
	var quantity float64 = 0

	lastFee := float64(0)

	for _, fill := range t.State.BuySideFills {
		quantity += fill.Quantity
		totalPrice += fill.Price * fill.Quantity
		if fill.CommissionAsset == "BNB" {
			cost += (fill.Price * fill.Quantity) * (1 + BNB_FEE)
			lastFee = BNB_FEE
		} else {
			cost += (fill.Price * fill.Quantity)
			lastFee = DEFAULT_FEE
			quantity = quantity - fill.CommissionAmount
		}
	}

	if quantity > 0 {
		t.State.AverageBuyPrice = round8(totalPrice / quantity)
		t.State.BuyFillQuantity = round8(quantity)
		t.State.BuyCost = round8(cost)
		t.State.EffectiveBuyPrice = round8(cost / quantity)

		// Use the fee from the most recent fee as the effective fee used for
		// calculation in profit and losses.
		t.State.Fee = lastFee
	}
}

func round8(val float64) float64 {
	return math.Round(val*100000000) / 100000000
}
