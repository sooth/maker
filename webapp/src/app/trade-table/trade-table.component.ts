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

import {Component, Input, OnChanges, OnInit} from "@angular/core";
import {MakerService, TradeMap, TradeState, TradeStatus} from "../maker.service";
import {Logger, LoggerService} from "../logger.service";
import {ToastrService} from "../toastr.service";
import {AggTrade} from '../binance-api.service';
import {Trade} from '../trade';
import {ActivatedRoute, Router} from "@angular/router";

@Component({
    selector: "app-trade-table",
    templateUrl: "./trade-table.component.html",
    styleUrls: ["./trade-table.component.scss"]
})
export class TradeTableComponent implements OnInit, OnChanges {

    TradeStatus = TradeStatus;

    trades: AppTradeState[] = [];

    private logger: Logger = null;

    @Input() showArchiveButtons: boolean = true;

    @Input() showTradeButtons: boolean = false;

    @Input() viewTrades: string = "all";

    private tradeMap: TradeMap = null;

    constructor(public maker: MakerService,
                private toastr: ToastrService,
                private router: Router,
                private route: ActivatedRoute,
                logger: LoggerService) {
        this.logger = logger.getLogger("TradeTableComponent");
    }

    ngOnInit() {
    }

    ngOnChanges() {
        this.renderTrades();
    }

    /**
     * Called by parent component to update the list of trades.
     *
     * This variation takes a map of trades keyed by ID.
     */
    onTradeMapUpdate(tradeMap: TradeMap) {
        this.tradeMap = tradeMap;
        this.renderTrades();
    }

    private renderTrades() {
        if (this.tradeMap == null) {
            return;
        }
        const trades: AppTradeState[] = [];
        for (const tradeId of Object.keys(this.tradeMap)) {
            trades.push(toAppTradeState(this.tradeMap[tradeId]));
        }
        this.trades = this.sortTrades(trades).filter((trade) => {
            switch (this.viewTrades) {
                case "open":
                    return trade.__isOpen;
                case "closed":
                    return !trade.__isOpen;
                default:
                    return true;
            }
        });
    }

    private sortTrades(trades: AppTradeState[]): AppTradeState[] {
        return trades.sort((a, b) => {
            return new Date(b.OpenTime).getTime() -
                new Date(a.OpenTime).getTime();
        });
    }

    onAggTrade(aggTrade: AggTrade) {
        for (const trade of this.trades) {
            if (trade.Symbol === aggTrade.symbol) {
                switch (trade.Status) {
                    case TradeStatus.DONE:
                    case TradeStatus.FAILED:
                    case TradeStatus.CANCELED:
                    case TradeStatus.ABANDONED:
                        break;
                    default:
                        trade.LastPrice = aggTrade.price;
                        Trade.updateProfit(trade, aggTrade.price);
                        trade.__rowClassName = getRowClass(trade);
                }
            }
        }
    }

    switchTradeView(what) {
        let params = Object.assign({}, this.route.snapshot.params);
        if (what === null) {
            delete (params["viewTrades"]);
        } else {
            params.viewTrades = what;
        }
        this.router.navigate([".", params], {
            queryParamsHandling: "merge",
        });
    }

    archive(trade: TradeState) {
        this.maker.archiveTrade(trade);
    }

    archiveAllClosed() {
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

}

export interface AppTradeState extends TradeState {
    __rowClassName?: string;
    __canArchive?: boolean;
    __canSell?: boolean;
    __canMarketSell?: boolean;
    __canAbandon?: boolean;
    __isOpen?: boolean;
    __canCancelSell?: boolean;

    /** The percent off from the purchase price. */
    buyPercentOffsetPercent?: number;
}

export function toAppTradeState(trade: TradeState): AppTradeState {
    const appTradeState = <AppTradeState>trade;
    appTradeState.__rowClassName = getRowClass(trade);
    appTradeState.__canArchive = canArchive(trade);
    appTradeState.__canSell = canSell(trade);
    appTradeState.__canMarketSell = canMarketSell(trade);
    appTradeState.__canCancelSell = canCancelSell(trade);
    appTradeState.__canAbandon = canAbandon(trade);
    appTradeState.__isOpen = Trade.isOpen(trade);
    return appTradeState;
}

export function canArchive(trade: AppTradeState): boolean {
    switch (trade.Status) {
        case TradeStatus.DONE:
        case TradeStatus.CANCELED:
        case TradeStatus.FAILED:
        case TradeStatus.ABANDONED:
            return true;
        default:
            return false;
    }
}

/**
 * Can a trade be sold. Within the scope of this application a sell order can
 * be placed for any non closed trade. If the buy has not been filled, the sell
 * order is queued until the buy is complete.
 *
 * The exception is market sell orders.
 */
export function canSell(trade: AppTradeState): boolean {
    switch (trade.Status) {
        case TradeStatus.NEW:
        case TradeStatus.PENDING_BUY:
        case TradeStatus.WATCHING:
        case TradeStatus.PENDING_SELL:
            return true;
        default:
            return false;
    }
}

/**
 * Can a trade be market sold.
 */
export function canMarketSell(trade: AppTradeState): boolean {
    switch (trade.Status) {
        case TradeStatus.WATCHING:
        case TradeStatus.PENDING_SELL:
            return true;
        default:
            return false;
    }
}

export function canCancelSell(trade: AppTradeState): boolean {
    switch (trade.Status) {
        case TradeStatus.ABANDONED:
        case TradeStatus.DONE:
        case TradeStatus.FAILED:
        case TradeStatus.CANCELED:
            return false;
        case TradeStatus.PENDING_SELL:
            return true;
        default:
            if (trade.LimitSell.Enabled) {
                return true;
            }
            return false;
    }
}

export function canAbandon(trade: AppTradeState): boolean {
    switch (trade.Status) {
        case TradeStatus.DONE:
        case TradeStatus.CANCELED:
        case TradeStatus.FAILED:
        case TradeStatus.ABANDONED:
            return false;
        default:
            return true;
    }
}

export function getRowClass(trade: AppTradeState): string {
    switch (trade.Status) {
        case TradeStatus.CANCELED:
        case TradeStatus.FAILED:
        case TradeStatus.ABANDONED:
            return "table-secondary";
        case TradeStatus.DONE:
            if (trade.ProfitPercent > 0) {
                return "bg-success";
            }
            return "bg-warning";
        case TradeStatus.NEW:
        case TradeStatus.PENDING_BUY:
            return "table-info";
        default:
            if (trade.ProfitPercent > 0) {
                return "table-success";
            }
            return "table-warning";
    }
}
