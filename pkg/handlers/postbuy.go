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

package handlers

import "gitlab.com/crankykernel/maker/pkg/maker"

type BuyOrderRequest struct {
	Symbol                  string              `json:"symbol"`
	Quantity                float64             `json:"quantity"`
	PriceSource             maker.PriceSource   `json:"priceSource"`
	LimitSellEnabled        bool                `json:"limitSellEnabled"`
	LimitSellType           maker.LimitSellType `json:"limitSellType"`
	LimitSellPercent        float64             `json:"limitSellPercent"`
	LimitSellPrice          float64             `json:"limitSellPrice"`
	StopLossEnabled         bool                `json:"stopLossEnabled"`
	StopLossPercent         float64             `json:"stopLossPercent"`
	TrailingProfitEnabled   bool                `json:"trailingProfitEnabled"`
	TrailingProfitPercent   float64             `json:"trailingProfitPercent"`
	TrailingProfitDeviation float64             `json:"trailingProfitDeviation"`
	Price                   float64             `json:"price"`
}

type BuyOrderResponse struct {
	TradeID string `json:"trade_id""`
}
