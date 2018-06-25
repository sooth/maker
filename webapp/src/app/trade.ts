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

import {AppTradeState} from './trade-table/trade-table.component';
import {TradeState, TradeStatus} from './maker.service';

export class Trade {

    static onLastPrice(trade: AppTradeState, lastPrice: number) {
        trade.LastPrice = lastPrice;
        Trade.updateProfit(trade, lastPrice);
    }

    /* TODO: Need to use sellable quantity. */
    static updateProfit(trade: AppTradeState, lastPrice: number) {
        if (trade.BuyFillQuantity > 0) {
            const profit = lastPrice * (1 - trade.Fee) - trade.EffectiveBuyPrice;
            trade.ProfitPercent = profit / trade.EffectiveBuyPrice * 100;
        }

        // Calculate the percent difference between our buy price and the
        // current price.
        const diffFromBuyPrice = trade.BuyOrder.Price - lastPrice;
        trade.buyPercentOffsetPercent = diffFromBuyPrice / lastPrice * 100;
    }

    static isOpen(trade: TradeState): boolean {
        switch (trade.Status) {
            case TradeStatus.DONE:
            case TradeStatus.FAILED:
            case TradeStatus.CANCELED:
            case TradeStatus.ABANDONED:
                return false;
            default:
                return true;
        }
    }

}

