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
	"gitlab.com/crankykernel/cryptotrader/binance"
	"time"
)

type OrderFill struct {
	Price            float64
	Quantity         float64
	CommissionAsset  string
	CommissionAmount float64
}

type HistoryType string

const (
	HistoryTypeCreated              HistoryType = "CREATED"
	HistoryTypeExecutionReport      HistoryType = "EXECUTION_REPORT"
	HistoryTypeSellOrder            HistoryType = "SELL_ORDER"
	HistoryTypeBuyCanceled          HistoryType = "BUY_CANCELED"
	HistoryTypeSellCanceled         HistoryType = "SELL_CANCELED"
	HistoryTypeTrailingProfitUpdate HistoryType = "TRAILING_PROFIT_UPDATE"
)

type HistoryEntry struct {
	Timestamp time.Time
	Type      HistoryType
	Fields    interface{}
}

type TradeState struct {
	Version int64

	// Trade ID local to this app. Its actually a ULID, but saved as a string.
	TradeID string

	History []HistoryEntry

	Symbol    string
	OpenTime  time.Time
	CloseTime *time.Time `json:",omitempty"`
	Status    TradeStatus
	Fee       float64

	BuyOrderId int64

	ClientOrderIDs map[string]bool

	BuyOrder struct {
		Quantity float64
		Price    float64
	}

	BuySideFills    []OrderFill `json:",omitempty"`
	BuyFillQuantity float64

	// The amount that can be sold. This is the quantity adjusted to any lot
	// size limitation like Binance's step size.
	SellableQuantity float64

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
		Type    LimitSellType
		Percent float64
		Price   float64
	}

	TrailingProfit struct {
		Enabled   bool
		Percent   float64
		Deviation float64
		Activated bool
		Price     float64
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
