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

import {
    AfterViewInit,
    Component,
    OnDestroy,
    OnInit,
    ViewChild
} from "@angular/core";
import {
    AccountInfo,
    AggTrade,
    Balance,
    BinanceApiService,
    PriceTicker
} from "../binance-api.service";
import {Observable} from "rxjs";
import {
    BinanceService,
    OpenTradeOptions,
    PriceSource
} from "../binance.service";
import {switchMap, tap} from "rxjs/operators";
import {Subscription} from "rxjs/Subscription";
import * as Mousetrap from "mousetrap";
import * as $ from "jquery";
import {round8, roundx} from "../utils";
import {ConfigService} from "../config.service";
import {MakerService, TradeMap} from "../maker.service";
import {Logger, LoggerService} from "../logger.service";
import {FormBuilder, FormGroup} from "@angular/forms";
import {ToastrService} from "../toastr.service";
import {TradeTableComponent} from '../trade-table/trade-table.component';

declare var window: any;

/**
 * The interface for the parts of the order form that should be saved between
 * trades and reloads.
 */
interface OrderFormSettingsModel {
    quoteAsset: string;
    symbol: string;
    priceSource: PriceSource;
    balancePercent: number;
    stopLossEnabled: boolean;
    stopLossPercent: number;
    trailingStopEnabled: boolean;
    trailingStopPercent: number;
    trailingStopDeviation: number;
    limitSellEnabled: boolean;
    limitSellPercent: number;
}

interface SavedState {
    orderFormSettings: OrderFormSettingsModel;
}

@Component({
    templateUrl: "./trade.component.html",
    styleUrls: ["./trade.component.scss"]
})
export class TradeComponent implements OnInit, OnDestroy, AfterViewInit {

    private localStorageKey = "binance.trade.component";

    BuyPriceSource = PriceSource;

    bidAskMap: { [key: string]: { bid: number, ask: number } } = {};

    orderFormSettings: OrderFormSettingsModel = {
        quoteAsset: "BTC",
        symbol: "ETHBTC",
        priceSource: PriceSource.BEST_BID,
        balancePercent: null,
        stopLossEnabled: false,
        stopLossPercent: 1,
        trailingStopEnabled: false,
        trailingStopPercent: 1,
        trailingStopDeviation: 0.25,
        limitSellEnabled: false,
        limitSellPercent: 0.1,
    };

    // Parts of the order form that don't persist.
    orderForm: {
        amount: number;
        quoteAmount: number;
        buyLimitPercent: number;
    } = {
        amount: null,
        quoteAmount: null,
        buyLimitPercent: null,
    };

    balances: { [key: string]: Balance } = {};

    private depthSubscription: Subscription = null;

    balancePercents: number[] = [];

    private subscriptions: Subscription[] = [];

    private logger: Logger = null;

    trailingStopForm: FormGroup;

    @ViewChild(TradeTableComponent) private tradeTable: TradeTableComponent;

    constructor(private api: BinanceApiService,
                public binance: BinanceService,
                private maker: MakerService,
                private config: ConfigService,
                private formBuilder: FormBuilder,
                private toastr: ToastrService,
                logger: LoggerService,
    ) {
        this.logger = logger.getLogger("trade-component");
    }

    ngOnInit() {
        this.config.loadConfig().subscribe(() => {
            this.balancePercents = this.config.getBalancePercents();
            this.orderFormSettings.balancePercent = this.balancePercents[0];
            this.reloadState();

            // Setup the trailing stop reactive form. It might be more work
            // than its worth for the reactive vs template form, at least for
            // this use case.
            this.trailingStopForm = this.formBuilder.group({
                enabled: [{
                    value: this.orderFormSettings.trailingStopEnabled,
                    disabled: false,
                }],
                percent: [{
                    value: this.orderFormSettings.trailingStopPercent,
                    disabled: !this.orderFormSettings.trailingStopEnabled,
                }],
                deviation: [{
                    value: this.orderFormSettings.trailingStopDeviation,
                    disabled: !this.orderFormSettings.trailingStopEnabled,
                }],
            });
            s = this.trailingStopForm.valueChanges.subscribe((data) => {
                this.orderFormSettings.trailingStopEnabled = data.enabled;
                this.orderFormSettings.trailingStopPercent = data.percent;
                this.orderFormSettings.trailingStopDeviation = data.deviation;
                if (data.enabled) {
                    this.trailingStopForm.controls.percent.enable({emitEvent: false});
                    this.trailingStopForm.controls.deviation.enable({emitEvent: false});
                } else {
                    this.trailingStopForm.controls.percent.disable({emitEvent: false});
                    this.trailingStopForm.controls.deviation.disable({emitEvent: false});
                }
                this.saveState();
            });
            this.subscriptions.push(s);
        });

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

    ngAfterViewInit() {
        this.binance.isReady$.subscribe(() => {
            let s = this.maker.onTradeUpdate.subscribe((tradeMap: TradeMap) => {
                setTimeout(() => {
                    this.tradeTable.onTradeMapUpdate(tradeMap);
                    console.log(this.tradeTable);
                }, 0);
            });
            this.subscriptions.push(s);

            s = this.maker.binanceAggTrades$.subscribe((aggTrade) => {
                setTimeout(() => {
                    this.tradeTable.onAggTrade(aggTrade);
                }, 0);
            });
            this.subscriptions.push(s);
        });
    }

    ngOnDestroy() {
        if (this.depthSubscription) {
            this.depthSubscription.unsubscribe();
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
        console.log("Saving state.");
        const state: SavedState = {
            orderFormSettings: this.orderFormSettings,
        };
        localStorage.setItem(this.localStorageKey, JSON.stringify(state));
    }

    private reloadState() {
        const rawState = localStorage.getItem(this.localStorageKey);
        if (!rawState) {
            this.logger.log("No saved state in local storage.");
            return;
        }
        try {
            const savedState: SavedState = JSON.parse(rawState);
            if (savedState.orderFormSettings) {
                Object.assign(this.orderFormSettings, savedState.orderFormSettings);
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
            this.orderFormSettings.symbol = symbol;
        } else {
            symbol = this.orderFormSettings.symbol;
        }

        if (!symbol) {
            this.saveState();
            return;
        }

        this.api.getPriceTicker(symbol)
                .subscribe((ticker: PriceTicker) => {
                    this.binance.lastPriceMap[ticker.symbol] = ticker.price;
                    this.updateOrderFormAssetAmount();
                });

        this.api.getBookTicker(symbol)
                .subscribe((ticker) => {
                    this.bidAskMap[symbol] = {
                        bid: ticker.bidPrice,
                        ask: ticker.askPrice,
                    };
                });

        /** TODO: This is just for the order info page, unsubscribe when done. */
        this.binance.subscribeToTradeStream(symbol);

        if (this.depthSubscription) {
            this.depthSubscription.unsubscribe();
        }
        this.depthSubscription = this.binance.subscribeToDepth(symbol)
                .subscribe((depth) => {
                    this.bidAskMap[depth.symbol] = {
                        bid: depth.bids[0].price,
                        ask: depth.asks[0].price,
                    };
                });

        this.saveState();
    }

    private onAggTrade(trade: AggTrade) {
        if (trade.symbol === this.orderFormSettings.symbol) {
            this.updateOrderFormAssetAmount();
        }
    }

    updateOrderFormAssetAmount() {
        if (!this.balances[this.orderFormSettings.quoteAsset]) {
            return;
        }
        const symbol = this.orderFormSettings.symbol;
        const available = this.balances[this.orderFormSettings.quoteAsset].free;
        const portion = round8(available * this.orderFormSettings.balancePercent / 100);
        const symbolInfo = this.binance.symbolMap[symbol];
        const stepSize = symbolInfo.stepSize;
        const lastTradePrice = this.binance.lastPriceMap[this.orderFormSettings.symbol];
        const amount = roundx(portion / lastTradePrice, 1 / stepSize);
        this.orderForm.quoteAmount = portion;
        this.orderForm.amount = amount;
    }

    makeOrder() {
        const options: OpenTradeOptions = {
            symbol: this.orderFormSettings.symbol,
            quantity: this.orderForm.amount,
            priceSource: this.orderFormSettings.priceSource,
            priceAdjustment: this.orderForm.buyLimitPercent,
            stopLossEnabled: this.orderFormSettings.stopLossEnabled,
            stopLossPercent: this.orderFormSettings.stopLossPercent,
            limitSellEnabled: this.orderFormSettings.limitSellEnabled,
            limitSellPercent: this.orderFormSettings.limitSellPercent,
            trailingStopEnabled: this.orderFormSettings.trailingStopEnabled,
            trailingStopPercent: this.orderFormSettings.trailingStopPercent,
            trailingStopDeviation: this.orderFormSettings.trailingStopDeviation,
        };
        this.binance.postBuyOrder(options).subscribe(() => {
        }, (error) => {
            console.log("failed to place order:");
            console.log(JSON.stringify(error));
            this.toastr.error(error.error.message, "Fail to make order");
        });
    }
}
