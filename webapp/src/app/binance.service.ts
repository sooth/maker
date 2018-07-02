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

import {Injectable} from "@angular/core";
import {
    AggTrade,
    BinanceApiService,
    buildTickerFromStream,
    Depth,
    ExchangeInfo,
    makeDepthFromStream,
    PriceTicker,
    SymbolInfo
} from "./binance-api.service";
import {roundx} from "./utils";
import {map, multicast, refCount, switchMap, take, tap} from "rxjs/operators";
import {Observable, Subject, Subscription} from "rxjs";
import {ReplaySubject} from "rxjs/ReplaySubject";
import {Logger, LoggerService} from "./logger.service";
import {HttpClient} from "@angular/common/http";
import {MakerService} from "./maker.service";
import {MakerApiService} from "./maker-api.service";
import {ToastrService} from "./toastr.service";

/**
 * Enum of types that can be used as a price source.
 */
export enum PriceSource {
    /** The price of the last trade is used. */
    LAST_PRICE = "LAST_PRICE",

    /** The best bid is used. Queries the REST API. */
    BEST_BID = "BEST_BID",

    /** The best ask is used. Queries the REST API. */
    BEST_ASK = "BEST_ASK",

    MANUAL = "MANUAL",
}

export enum LimitSellType {
    PERCENT = "PERCENT",
    PRICE = "PRICE",
}

export interface SymbolMap {
    [key: string]: SymbolInfo;
}

@Injectable()
export class BinanceService {

    /** A list of all symbols (pairs). */
    symbols: string[] = [];

    /** A list of all quote assets: BTC, BNB, etc. */
    quoteAssets: string[] = [];

    /**
     * SymbolInfo objects keyed by symbol.
     */
    symbolMap: SymbolMap = {};

    /**
     * A symbol map of symbol to last price.
     */
    lastPriceMap: { [key: string]: number } = {};

    aggTradeStream$: Subject<AggTrade> = new Subject();

    private aggTradeStreams: { [key: string]: Subscription } = {};

    streams$: { [key: string]: Observable<any> } = {};

    private isReadySubject: Subject<boolean> = null;
    public isReady$: Observable<boolean> = null;

    private logger: Logger = null;

    constructor(private api: BinanceApiService,
                private maker: MakerService,
                private makerApi: MakerApiService,
                private toastr: ToastrService,
                http: HttpClient,
                logger: LoggerService) {

        this.logger = logger.getLogger("binance.service");

        this.isReadySubject = new ReplaySubject<boolean>(1);
        this.isReady$ = this.isReadySubject.pipe(take(1));

        // Get config then do initialization that depends on config.
        this.makerApi.getConfig().subscribe((config) => {
            this.api.apiKey = config["binance.api.key"];
            this.api.apiSecret = config["binance.api.secret"];

            this.updateExchangeInfo()
                    .pipe(switchMap(() => {
                        return this.api.getPriceTicker("BTCUSDT")
                                .pipe(tap((ticker) => {
                                    this.lastPriceMap[ticker.symbol] = ticker.price;
                                }));
                    })).subscribe(() => {
                this.isReadySubject.next(true);
            });
        });

        this.maker.binanceAggTrades$.subscribe((trade) => {
            this.onAggTrade(trade);
        });

        // Subscribe to the aggTrade stream, even though we publish to it.
        this.aggTradeStream$.subscribe((trade) => {
            this.onAggTrade(trade);
        });

        // Subscribe to the BTCUSDT ticker to assign a USD value to trades.
        this.subscribeToTicker("BTCUSDT").subscribe((ticker) => {
            this.lastPriceMap["BTCUSDT"] = ticker.price;
        });
    }

    private onAggTrade(trade: AggTrade) {
        this.lastPriceMap[trade.symbol] = trade.price;
    }

    postBuyOrder(body: OpenTradeOptions) {
        return this.api.postBuyOrder(body);
    }

    /**
     * Adjust the price by a percentage, up or down, taking into account the
     * symbols tick size.
     */
    adjustPrice(symbol: string, price: number, percent: number): number {
        if (percent === 0) {
            return price;
        }
        const limitPrice = price * (1 + (percent / 100));
        const symbolInfo = this.symbolMap[symbol];
        const tickSize = symbolInfo.tickSize;
        const adjustedPrice = roundx(limitPrice, 1 / tickSize);
        return adjustedPrice;
    }

    subscribeToTradeStream(symbol: string) {
        symbol = symbol.toLowerCase();
        if (symbol in this.aggTradeStreams) {
            return;
        }
        const streams = [
            `${symbol}@aggTrade`,
        ];
        this.aggTradeStreams[symbol] = this.api.openMultiStream(streams)
                .subscribe((message) => {
                    if (message.streamType === "aggTrade") {
                        this.aggTradeStream$.next(message.getAggTrade());
                    }
                });
    }

    subscribeToTicker(symbol: string): Observable<PriceTicker> {
        const stream = `${symbol.toLowerCase()}@ticker`;
        if (!this.streams$[stream]) {
            this.logger.log(`Creating new ticker stream for ${symbol}.`);
            const path = `/ws/${stream}`;
            this.streams$[stream] = this.api.openStream(path)
                    .pipe(
                            multicast(new Subject<any>()),
                            refCount(),
                            map((ticker) => {
                                return buildTickerFromStream(ticker);
                            })
                    );
        }
        return this.streams$[stream];
    }

    subscribeToDepth(symbol: string): Observable<Depth> {
        const stream = `${symbol.toLowerCase()}@depth5`;
        if (!this.streams$[stream]) {
            const path = `/ws/${stream}`;
            this.streams$[stream] = this.api.openStream(path)
                    .pipe(
                            multicast(new Subject<any>()),
                            refCount(),
                            map((depth) => {
                                return makeDepthFromStream(symbol, depth);
                            })
                    );
        }
        return this.streams$[stream];
    }

    updateExchangeInfo(): Observable<ExchangeInfo> {
        return this.api.getExchangeInfo().pipe(tap((exchangeInfo) => {
            const quoteAssets: any = {};

            exchangeInfo.symbols.forEach((symbol) => {
                this.symbolMap[symbol.symbol] = symbol;
                quoteAssets[symbol.quoteAsset] = true;
            });

            this.symbols = Object.keys(this.symbolMap).filter((key) => {
                if (this.symbolMap[key].status !== "TRADING") {
                    return false;
                }
                return true;
            }).sort();

            this.quoteAssets = Object.keys(quoteAssets);
        }));
    }

}

export interface OpenTradeOptions {
    symbol: string;
    quantity: number;

    priceSource: PriceSource;
    priceAdjustment: number;
    price?: number;

    stopLossEnabled?: boolean;
    stopLossPercent?: number;

    limitSellEnabled?: boolean;
    limitSellType?: LimitSellType;
    limitSellPercent?: number;
    limitSellPrice?: number;

    trailingProfitEnabled?: boolean;
    trailingProfitPercent?: number;
    trailingProfitDeviation?: number;
}
