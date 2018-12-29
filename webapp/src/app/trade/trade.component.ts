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

import {AfterViewInit, Component, OnDestroy, OnInit, ViewChild} from "@angular/core";
import {AccountInfo, AggTrade, Balance, BinanceApiService, PriceTicker} from "../binance-api.service";
import {Observable} from "rxjs";
import {BinanceService, LimitSellType, OpenTradeOptions, PriceSource} from "../binance.service";
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
import {ActivatedRoute, Router} from '@angular/router';

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
    trailingProfitEnabled: boolean;
    trailingProfitPercent: number;
    trailingProfitDeviation: number;
    limitSellEnabled: boolean;
    limitSellPercent: number;
    limitSellPriceEnabled: boolean;
}

interface SavedState {
    orderFormSettings: OrderFormSettingsModel;
}

@Component({
    templateUrl: "./trade.component.html",
    styleUrls: ["./trade.component.scss"]
})
export class TradeComponent implements OnInit, OnDestroy, AfterViewInit {

    OFFSET_TYPE_NONE = "NONE";
    OFFSET_TYPE_ABSOLUTE = "ABSOLUTE";

    private localStorageKey = "binance.trade.component";

    BuyPriceSource = PriceSource;

    bidAskMap: {
        [key: string]: {
            bid: number,
            ask: number
        }
    } = {};

    lastPriceMap: {
        [key: string]: number,
    } = {};

    orderFormSettings: OrderFormSettingsModel = {
        quoteAsset: "BTC",
        symbol: "ETHBTC",
        priceSource: PriceSource.BEST_BID,
        balancePercent: null,
        stopLossEnabled: false,
        stopLossPercent: 1,
        trailingProfitEnabled: false,
        trailingProfitPercent: 1,
        trailingProfitDeviation: 0.25,
        limitSellEnabled: false,
        limitSellPercent: 0.1,
        limitSellPriceEnabled: false,
    };

    // Parts of the order form that don't persist.
    orderForm: {
        amount: number;
        quoteAmount: number;
        buyLimitPercent: number;
        manualPrice: string;
        limitSellPrice: string;

        offsetType: string;
        offsetValue: number;
        offsetPercent: number;
    } = {
        amount: null,
        quoteAmount: null,
        buyLimitPercent: null,
        manualPrice: null,
        limitSellPrice: null,

        offsetType: this.OFFSET_TYPE_NONE,
        offsetValue: 0.0,
        offsetPercent: 0,
    };

    balances: { [key: string]: Balance } = {};

    private depthSubscription: Subscription = null;

    private tradeSubscription: Subscription = null;

    balancePercents: number[] = [];

    private subs: Subscription[] = [];

    private logger: Logger = null;

    trailingProfitForm: FormGroup;

    priceStepSize: number = 0.00000001;

    @ViewChild(TradeTableComponent) private tradeTable: TradeTableComponent;

    constructor(private api: BinanceApiService,
                public binance: BinanceService,
                private binanceApi: BinanceApiService,
                private maker: MakerService,
                private config: ConfigService,
                private formBuilder: FormBuilder,
                private toastr: ToastrService,
                private route: ActivatedRoute,
                private router: Router,
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
            this.trailingProfitForm = this.formBuilder.group({
                enabled: [{
                    value: this.orderFormSettings.trailingProfitEnabled,
                    disabled: false,
                }],
                percent: [{
                    value: this.orderFormSettings.trailingProfitPercent,
                    disabled: !this.orderFormSettings.trailingProfitEnabled,
                }],
                deviation: [{
                    value: this.orderFormSettings.trailingProfitDeviation,
                    disabled: !this.orderFormSettings.trailingProfitEnabled,
                }],
            });
            let s = this.trailingProfitForm.valueChanges.subscribe((data) => {
                this.orderFormSettings.trailingProfitEnabled = data.enabled;
                if (data.percent != undefined) {
                    this.orderFormSettings.trailingProfitPercent = data.percent;
                }
                if (data.percent != undefined) {
                    this.orderFormSettings.trailingProfitDeviation = data.deviation;
                }
                if (data.enabled) {
                    this.trailingProfitForm.controls.percent.enable({emitEvent: false});
                    this.trailingProfitForm.controls.deviation.enable({emitEvent: false});
                } else {
                    this.trailingProfitForm.controls.percent.disable({emitEvent: false});
                    this.trailingProfitForm.controls.deviation.disable({emitEvent: false});
                }
                this.saveState();
            });
            this.subs.push(s);
        });

        this.binance.isReady$.pipe(switchMap(() => {
            return this.updateAccountInfo();
        })).subscribe(() => {
            this.changeSymbol();
        }, () => {
            // Error.
        }, () => {
        });

        if (this.route.snapshot.params.symbol) {
            this.orderFormSettings.symbol = this.route.snapshot.params.symbol;
            console.log(`Initialized symbol to ${this.orderFormSettings.symbol}`);
        }

        this.binance.isReady$.subscribe(() => {
            this.route.params.subscribe((params) => {
                const newSymbol = params.symbol;
                if (newSymbol && newSymbol != this.orderFormSettings.symbol) {
                    this.changeSymbol(newSymbol);
                }
            })
        });

        let s = this.maker.binanceAccountInfo$.subscribe((accountInfo) => {
            this.updateBalances(accountInfo);
        });
        this.subs.push(s);

        Mousetrap.bind("/", () => {
            window.scrollTo(0, 0);
            $("#symbolInput").focus();
        });

    }

    ngAfterViewInit() {
        this.binance.isReady$.subscribe(() => {
            let s = this.maker.tradeMap$.subscribe((tradeMap: TradeMap) => {
                setTimeout(() => {
                    this.tradeTable.onTradeMapUpdate(tradeMap);
                }, 0);
            });
            this.subs.push(s);

            s = this.maker.binanceAggTrades$.subscribe((aggTrade) => {
                setTimeout(() => {
                    this.tradeTable.onAggTrade(aggTrade);
                }, 0);
            });
            this.subs.push(s);
        });
    }

    ngOnDestroy() {
        if (this.depthSubscription) {
            this.depthSubscription.unsubscribe();
        }
        if (this.tradeSubscription) {
            this.tradeSubscription.unsubscribe();
        }
        for (const sub of this.subs) {
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

        this.router.navigate(["/trade", {symbol: symbol}])

        if (!symbol) {
            this.saveState();
            return;
        }

        this.api.getPriceTicker(symbol)
            .subscribe((ticker: PriceTicker) => {
                this.lastPriceMap[ticker.symbol] = ticker.price;
                this.updateOrderFormAssetAmount();
                this.orderForm.manualPrice = ticker.price.toFixed(8);
                this.orderForm.limitSellPrice = ticker.price.toFixed(8);
            });
        this.priceStepSize = this.binance.symbolMap[symbol].tickSize;

        this.api.getBookTicker(symbol)
            .subscribe((ticker) => {
                this.bidAskMap[symbol] = {
                    bid: ticker.bidPrice,
                    ask: ticker.askPrice,
                };
            });

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

        if (this.tradeSubscription) {
            this.tradeSubscription.unsubscribe();
        }
        this.tradeSubscription = this.binance.subscribeAggTradeStream(symbol)
            .subscribe((aggTrade) => {
                this.lastPriceMap[symbol] = aggTrade.price;
                this.onAggTrade(aggTrade);
            });

        this.saveState();
    }

    syncManualPrice() {
        this.orderForm.manualPrice = this.lastPriceMap[this.orderFormSettings.symbol].toFixed(8);
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
        const lastTradePrice = this.lastPriceMap[this.orderFormSettings.symbol];
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
            trailingProfitEnabled: this.orderFormSettings.trailingProfitEnabled,
            trailingProfitPercent: this.orderFormSettings.trailingProfitPercent,
            trailingProfitDeviation: this.orderFormSettings.trailingProfitDeviation,
            price: +this.orderForm.manualPrice,
        };

        if (this.orderFormSettings.priceSource != PriceSource.MANUAL) {
            if (this.orderForm.offsetType == this.OFFSET_TYPE_ABSOLUTE) {
                options.offsetValue = +this.orderForm.offsetValue;
            }
        }

        if (this.orderFormSettings.limitSellEnabled) {
            options.limitSellEnabled = true;
            options.limitSellType = LimitSellType.PERCENT;
            options.limitSellPercent = this.orderFormSettings.limitSellPercent;
        } else if (this.orderFormSettings.limitSellPriceEnabled) {
            options.limitSellEnabled = true;
            options.limitSellType = LimitSellType.PRICE;
            options.limitSellPrice = +this.orderForm.limitSellPrice;
        }

        this.binance.postBuyOrder(options).subscribe(() => {
        }, (error) => {
            console.log("failed to place order:");
            console.log(JSON.stringify(error));
            this.toastr.error(error.error.message, "Fail to make order");
        });
    }

    toggleLimitSellType(type: string) {
        if (type == 'PERCENT') {
            if (this.orderFormSettings.limitSellPriceEnabled) {
                this.orderFormSettings.limitSellPriceEnabled = false;
            }
            this.orderFormSettings.limitSellEnabled = !this.orderFormSettings.limitSellEnabled;
        } else if (type == 'PRICE') {
            this.orderFormSettings.limitSellEnabled = false;
            this.orderFormSettings.limitSellPriceEnabled = !this.orderFormSettings.limitSellPriceEnabled;
        }

        if (this.orderFormSettings.limitSellPriceEnabled) {
            this.orderForm.limitSellPrice =
                this.lastPriceMap[this.orderFormSettings.symbol].toFixed(8);
        }

    }

    onManualPriceInput() {
        this.orderForm.manualPrice = this.toFixed(this.orderForm.manualPrice, 8);
    }

    toFixed(value: number | string, fractionDigits: number): string {
        return (+value).toFixed(fractionDigits);
    }

    private getPrice(): number {
        switch (this.orderFormSettings.priceSource) {
            case PriceSource.MANUAL:
                return +this.orderForm.manualPrice;
            case PriceSource.LAST_PRICE:
                return this.lastPriceMap[this.orderFormSettings.symbol];
            case PriceSource.BEST_BID:
                return this.bidAskMap[this.orderFormSettings.symbol].bid;
            case PriceSource.BEST_ASK:
                return this.bidAskMap[this.orderFormSettings.symbol].ask;
            default:
                break;
        }
        return 0;
    }
}
