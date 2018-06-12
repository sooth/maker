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

import {Component, OnDestroy, OnInit} from "@angular/core";
import {AccountInfo, AggTrade, Balance, BinanceApiService, PriceTicker} from "../binance-api.service";
import {Observable} from "rxjs";
import {BinanceService, PriceSource, TradeOptions} from "../binance.service";
import {switchMap, tap} from "rxjs/operators";
import {Subscription} from "rxjs/Subscription";
import * as Mousetrap from "mousetrap";
import * as $ from "jquery";
import {round8, roundx} from "../utils";
import {ConfigService} from "../config.service";
import {MakerService} from "../maker.service";

declare var window: any;

interface SavedState {
    model: any;
}

@Component({
    templateUrl: "./trade.component.html",
    styleUrls: ["./trade.component.scss"]
})
export class TradeComponent implements OnInit, OnDestroy {

    private localStorageKey = "binance.trade.component";

    BuyPriceSource = PriceSource;

    bidAskMap: { [key: string]: { bid: number, ask: number } } = {};

    model: {
        quoteAsset: string;
        symbol: string;
        orderInput: {
            amount: number;
            quoteAmount: number;
            buyLimitPercent: number;
            priceSource: PriceSource;

            stopLossEnabled: boolean;
            stopLossPercent: number;

            trailingStopEnabled: boolean;
            trailingStopPercent: number;
            trailingStopDeviation: number;

            // Quick sell parameters. If turned on, a limit sell will be placed
            // immediately on fill.
            limitSellEnabled: boolean;
            limitSellPercent: number;

            quoteAssetPercent: number;
        };
    } = {
        quoteAsset: "BTC",
        symbol: "ETHBTC",
        orderInput: {
            amount: null,
            quoteAmount: null,
            buyLimitPercent: null,
            priceSource: PriceSource.BEST_BID,

            stopLossEnabled: false,
            stopLossPercent: 1,

            trailingStopEnabled: false,
            trailingStopPercent: 1,
            trailingStopDeviation: 0.25,

            limitSellEnabled: false,
            limitSellPercent: 0.1,

            quoteAssetPercent: 10,
        },
    };

    balances: { [key: string]: Balance } = {};

    private depthSubscription: Subscription = null;

    private userDataStreamSubscription: Subscription = null;

    balancePercents: number[] = [];

    private subscriptions: Subscription[] = [];

    constructor(private api: BinanceApiService,
                public binance: BinanceService,
                private maker: MakerService,
                private config: ConfigService,
    ) {
    }

    ngOnInit() {
        this.config.loadConfig().subscribe(() => {
            this.balancePercents = this.config.getBalancePercents();
        });

        this.reloadState();

        this.binance.isReady$.pipe(switchMap(() => {
            return this.updateAccountInfo();
        })).subscribe(() => {
            this.changeSymbol();
        }, () => {
            // Error.
        }, () => {
        });

        let s = this.maker.binanceAccountInfo$.subscribe((accountInfo) => {
            this.updateBalances(accountInfo);
        });
        this.subscriptions.push(s);

        s = this.binance.aggTradeStream$.subscribe((trade) => {
            this.onAggTrade(trade);
        });
        this.subscriptions.push(s);

        Mousetrap.bind("/", () => {
            window.scrollTo(0, 0);
            $("#symbolInput").focus();
        });
    }

    ngOnDestroy() {
        if (this.depthSubscription) {
            this.depthSubscription.unsubscribe();
        }
        if (this.userDataStreamSubscription) {
            this.userDataStreamSubscription.unsubscribe();
        }
        for (const sub of this.subscriptions) {
            sub.unsubscribe();
        }
    }

    private updateAccountInfo(): Observable<AccountInfo> {
        return this.api.getAccountInfo().pipe(tap((accountInfo) => {
            console.log("Updating account info.");
            this.updateBalances(accountInfo);
        }));
    }

    private saveState() {
        const state: SavedState = {
            model: this.model,
        };
        localStorage.setItem(this.localStorageKey, JSON.stringify(state));
    }

    private reloadState() {
        const rawState = localStorage.getItem(this.localStorageKey);
        if (!rawState) {
            return;
        }
        try {
            const savedState: SavedState = JSON.parse(rawState);
            if (savedState.model) {
                Object.assign(this.model, savedState.model);
            }
        } catch (err) {
            console.log("error: failed to restore saved status:");
            console.log(err);
        }
    }

    private updateBalances(accountInfo: AccountInfo) {
        for (const balance of accountInfo.balances) {
            this.balances[balance.asset] = balance;
        }
    }

    changeQuoteAsset() {
        this.binance.updateExchangeInfo().subscribe();
        this.saveState();
    }

    changeSymbol(symbol: string = null) {
        if (symbol != null) {
            this.model.symbol = symbol;
        } else {
            symbol = this.model.symbol;
        }
        if (!this.model.symbol) {
            this.saveState();
            return;
        }
        this.api.getPriceTicker(this.model.symbol)
                .subscribe((ticker: PriceTicker) => {
                    this.binance.lastPriceMap[ticker.symbol] = ticker.price;
                    this.updateOrderFormAssetAmount();
                });

        this.api.getBookTicker(this.model.symbol)
                .subscribe((ticker) => {
                    this.bidAskMap[symbol] = {
                        bid: ticker.bidPrice,
                        ask: ticker.askPrice,
                    };
                });

        /** TODO: This is just for the order info page, unsubscribe when done. */
        this.binance.subscribeToTradeStream(this.model.symbol);

        if (this.depthSubscription) {
            this.depthSubscription.unsubscribe();
        }
        this.depthSubscription = this.binance.subscribeToDepth(this.model.symbol)
                .subscribe((depth) => {
                    this.bidAskMap[depth.symbol] = {
                        bid: depth.bids[0].price,
                        ask: depth.asks[0].price,
                    };
                });

        this.saveState();
    }

    private onAggTrade(trade: AggTrade) {
        if (trade.symbol === this.model.symbol) {
            this.updateOrderFormAssetAmount();
        }
    }

    updateOrderFormAssetAmount() {
        if (!this.balances[this.model.quoteAsset]) {
            return;
        }
        const symbol = this.model.symbol;
        const available = this.balances[this.model.quoteAsset].free;
        const portion = round8(available * this.model.orderInput.quoteAssetPercent / 100);
        const symbolInfo = this.binance.symbolMap[symbol];
        const stepSize = symbolInfo.stepSize;
        const lastTradePrice = this.binance.lastPriceMap[this.model.symbol];
        const amount = roundx(portion / lastTradePrice, 1 / stepSize);
        this.model.orderInput.quoteAmount = portion;
        this.model.orderInput.amount = amount;
    }

    makeOrder() {
        const options: TradeOptions = {
            symbol: this.model.symbol,
            quantity: this.model.orderInput.amount,
            priceSource: this.model.orderInput.priceSource,
            priceAdjustment: this.model.orderInput.buyLimitPercent,
            stopLossEnabled: this.model.orderInput.stopLossEnabled,
            stopLossPercent: this.model.orderInput.stopLossPercent,
            limitSellEnabled: this.model.orderInput.limitSellEnabled,
            limitSellPercent: this.model.orderInput.limitSellPercent,
            trailingStopEnabled: this.model.orderInput.trailingStopEnabled,
            trailingStopPercent: this.model.orderInput.trailingStopPercent,
            trailingStopDeviation: this.model.orderInput.trailingStopDeviation,
        };
        this.binance.openTrade(options);
    }
}
