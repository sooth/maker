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

func TradeV0ToTradeV1(old TradeStateV0) TradeState {
	state := TradeState{}
	state.Version = TRADE_STATE_VERSION
	state.TradeID = old.LocalID
	state.Symbol = old.Symbol
	state.OpenTime = old.OpenTime
	state.CloseTime = old.CloseTime
	state.Status = old.Status
	state.Fee = old.Fee
	state.BuyOrderId = old.BuyOrderId
	state.ClientOrderIDs = old.ClientOrderIDs
	state.BuyOrder = old.BuyOrder
	state.BuySideFills = old.BuySideFills
	state.BuyFillQuantity = old.BuyFillQuantity
	state.AverageBuyPrice = old.AverageBuyPrice
	state.BuyCost = old.BuyCost
	state.EffectiveBuyPrice = old.EffectiveBuyPrice
	state.SellOrderId = old.SellOrderId
	state.SellSideFills = old.SellSideFills
	state.SellFillQuantity = old.SellFillQuantity
	state.AverageSellPrice = old.AverageSellPrice
	state.SellCost = old.SellCost
	state.StopLoss = old.StopLoss
	state.LimitSell = old.LimitSell
	state.TrailingProfit = old.TrailingProfit
	state.Profit = old.Profit
	state.ProfitPercent = old.ProfitPercent
	state.LastBuyStatus = old.LastBuyStatus
	state.SellOrder = old.SellOrder
	state.LastPrice = old.LastPrice
	return state
}
