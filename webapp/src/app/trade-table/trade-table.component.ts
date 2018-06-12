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

import {Component, EventEmitter, OnInit, Output} from '@angular/core';
import {MakerService, TradeMap, TradeState, TradeStatus} from '../maker.service';
import {Logger, LoggerService} from '../logger.service';
import {BinanceService} from '../binance.service';
import {AggTrade} from '../binance-api.service';

@Component({
    selector: 'app-trade-table',
    templateUrl: './trade-table.component.html',
    styleUrls: ['./trade-table.component.scss']
})
export class TradeTableComponent implements OnInit {

    TradeStatus = TradeStatus;

    trades: AppTradeState[] = [];

    private logger: Logger = null;

    showJson = false;

    @Output() symbolClickHandler: EventEmitter<any> = new EventEmitter();

    constructor(public maker: MakerService,
                logger: LoggerService,
                private binance: BinanceService) {
        this.logger = logger.getLogger('TradeTableComponent');
    }

    ngOnInit() {
        this.maker.onTradeUpdate.subscribe((tradeMap: TradeMap) => {
            const trades: AppTradeState[] = [];
            for (const tradeId of Object.keys(tradeMap)) {
                const trade: TradeState = tradeMap[tradeId];
                switch (trade.Status) {
                    case TradeStatus.DONE:
                    case TradeStatus.CANCELED:
                    case TradeStatus.FAILED:
                    default:
                        const appTradeState = <AppTradeState>trade;
                        appTradeState.lastPrice = this.binance.lastPriceMap[trade.Symbol];
                        this.updateProfit(trade, this.binance.lastPriceMap[trade.Symbol]);
                        appTradeState.__rowClassName = this.getRowClass(trade);
                        appTradeState.__canArchive = this.getCanArchive(trade);
                        trades.push(appTradeState);
                        break;
                }
            }
            this.trades = trades.sort((a, b) => {
                return new Date(b.OpenTime).getTime() -
                        new Date(a.OpenTime).getTime();
            });
        });

        this.maker.binanceAggTrades$.subscribe((aggTrade) => {
            for (const trade of this.trades) {
                if (trade.Symbol === aggTrade.symbol) {
                    switch (trade.Status) {
                        case TradeStatus.DONE:
                        case TradeStatus.FAILED:
                        case TradeStatus.CANCELED:
                            break;
                        default:
                            trade.lastPrice = aggTrade.price;
                            this.updateProfit(trade, aggTrade.price);
                            trade.__rowClassName = this.getRowClass(trade);
                    }
                }
            }
        });
    }

    onSymbolClick(symbol: string) {
        this.symbolClickHandler.emit(symbol);
    }

    private getCanArchive(trade: AppTradeState): boolean {
        switch (trade.Status) {
            case TradeStatus.DONE:
            case TradeStatus.CANCELED:
            case TradeStatus.FAILED:
                return true;
            default:
                return false;
        }
    }

    getRowClass(trade: AppTradeState): string {
        switch (trade.Status) {
            case TradeStatus.CANCELED:
            case TradeStatus.FAILED:
                return 'table-secondary';
            case TradeStatus.DONE:
                if (trade.ProfitPercent > 0) {
                    return 'bg-success';
                }
                return 'bg-warning';
            case TradeStatus.NEW:
            case TradeStatus.PENDING_BUY:
                return 'table-info';
            default:
                if (trade.ProfitPercent > 0) {
                    return 'table-success';
                }
                return 'table-warning';
        }
    }

    cancelBuy(trade: TradeState) {
        this.maker.cancelBuy(trade);
    }

    cancelSell(trade: TradeState) {
        this.maker.cancelSell(trade);
    }

    limitSell(trade: TradeState, percent: number) {
        this.maker.limitSell(trade, percent);
    }

    marketSell(trade: TradeState) {
        this.maker.marketSell(trade);
    }

    archive(trade: TradeState) {
        this.maker.archiveTrade(trade);
    }

    archiveAll() {
        for (const trade of this.trades) {
            switch (trade.Status) {
                case TradeStatus.FAILED:
                case TradeStatus.CANCELED:
                case TradeStatus.DONE:
                    this.archive(trade);
                    break;
            }
        }
    }

    archiveCanceledFailed() {
        for (const trade of this.trades) {
            switch (trade.Status) {
                case TradeStatus.FAILED:
                case TradeStatus.CANCELED:
                    this.archive(trade);
                    break;
            }
        }
    }

    private updateProfit(trade: AppTradeState, lastPrice: number) {
        if (trade.BuyFillQuantity > 0) {
            const profit = lastPrice * (1 - trade.Fee) - trade.EffectiveBuyPrice;
            trade.ProfitPercent = profit / trade.EffectiveBuyPrice * 100;
        }

        // Calculate the percent different between our buy price and the
        // current price.
        const diffFromBuyPrice = trade.BuyOrder.Price - lastPrice;
        trade.buyPercentOffsetPercent = diffFromBuyPrice / lastPrice * 100;
    }
}

export interface AppTradeState extends TradeState {
    lastPrice?: number;
    __rowClassName?: string;
    __canArchive?: boolean;

    /** The percent off from the purchase price. */
    buyPercentOffsetPercent?: number;
}
