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

package maker

import (
	"time"
	"math"
	"github.com/crankykernel/cryptotrader/binance"
)

const DEFAULT_FEE = float64(0.001)
const BNB_FEE = float64(0.0005)

type TradeStatus string

const (
	TradeStatusNew         TradeStatus = "NEW"
	TradeStatusFailed      TradeStatus = "FAILED"
	TradeStatusPendingBuy  TradeStatus = "PENDING_BUY"
	TradeStatusWatching    TradeStatus = "WATCHING"
	TradeStatusPendingSell TradeStatus = "PENDING_SELL"
	TradeStatusDone        TradeStatus = "DONE"
	TradeStatusCanceled    TradeStatus = "CANCELED"
	TradeStatusAbandoned   TradeStatus = "ABANDONED"
)

type BuyOrder struct {
	Quantity float64
	Price    float64
}

type OrderFill struct {
	Price            float64
	Quantity         float64
	CommissionAsset  string
	CommissionAmount float64
}

type TradeState struct {
	// Trade ID local to this app. Its actually a ULID, but saved as a string.
	LocalID string

	Symbol    string
	OpenTime  time.Time
	CloseTime *time.Time `json:",omitempty"`
	Status    TradeStatus
	Fee       float64

	BuyOrderId int64

	ClientOrderIDs map[string]bool

	BuyOrder BuyOrder

	BuySideFills    []OrderFill `json:",omitempty"`
	BuyFillQuantity float64

	// The average buy price per unit not accounting for fees.
	AverageBuyPrice float64

	// The total cost of the buy, including fees.
	BuyCost float64

	// The buy price per unit accounting for fees.
	EffectiveBuyPrice float64

	SellOrderId int64

	SellSideFills    []OrderFill `json:",omitempty"`
	SellFillQuantity float64
	AverageSellPrice float64
	SellCost         float64

	StopLoss struct {
		Enabled   bool
		Percent   float64
		Triggered bool
	}

	LimitSell struct {
		Enabled bool
		Percent float64
	}

	TrailingStop struct {
		Enabled   bool
		Percent   float64
		Deviation float64
		Activated bool
		Price     float64
		// Set to true when the sell order has been sent.
		Triggered bool
	}

	// The profit in units of the quote asset.
	Profit float64

	// The profit as a percentage (0-100).
	ProfitPercent float64

	LastBuyStatus binance.OrderStatus

	SellOrder struct {
		Status   binance.OrderStatus
		Type     string
		Quantity float64
		Price    float64
	}

	// The last known price for this symbol. Use to estimate profit. Source may
	// not always be the last price, but could also be the last best bid or ask.
	LastPrice float64
}

type Trade struct {
	State TradeState
}

func NewTrade() *Trade {
	return &Trade{
		State: TradeState{
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

func (t *Trade) SetLimitSell(enable bool, percent float64) {
	t.State.LimitSell.Enabled = enable
	t.State.LimitSell.Percent = percent
}

func (t *Trade) SetStopLoss(enable bool, percent float64) {
	t.State.StopLoss.Enabled = enable
	t.State.StopLoss.Percent = percent
}

func (t *Trade) SetTrailingStop(enable bool, percent float64, deviation float64) {
	t.State.TrailingStop.Enabled = enable
	t.State.TrailingStop.Percent = percent
	t.State.TrailingStop.Deviation = deviation
}

func (t *Trade) AddBuyFill(report binance.StreamExecutionReport) {
	fill := OrderFill{
		Price:            report.LastExecutedPrice,
		Quantity:         report.LastExecutedQuantity,
		CommissionAmount: report.CommissionAmount,
		CommissionAsset:  report.CommissionAsset,
	}
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
			cost += (fill.Price * fill.Quantity) * (1 - DEFAULT_FEE)
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
			cost += (fill.Price * fill.Quantity) * (1 + DEFAULT_FEE)
			lastFee = DEFAULT_FEE
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
