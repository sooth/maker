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
import {Observable} from "rxjs";
import {Observer} from "rxjs/Observer";

const STREAM_ROOT = "wss://stream.binance.com:9443";

@Injectable()
export class BinanceApiService {
    openStream(path: string): Observable<any> {
        const url = `${STREAM_ROOT}${path}`;
        return makeWebSocketObservable(url);
    }
}

function makeWebSocketObservable(url: string): Observable<any> {
    return Observable.create((observer: Observer<any>) => {

        let ws: WebSocket = null;
        let closeRequested = false;

        const openWebSocket = () => {
            console.log(`websocket: connecting to ${url}.`);
            ws = new WebSocket(url);

            ws.onmessage = (event) => {
                observer.next(JSON.parse(event.data));
            };

            ws.onerror = (event) => {
                console.log(`websocket: error: ${url}: ${JSON.stringify(event)}`);
                console.log(event);
                observer.error(event);
            };

            ws.onclose = () => {
                console.log(`websocket: closed ${url}.`);
                if (!closeRequested) {
                    openWebSocket();
                }
            };
        };

        openWebSocket();

        return () => {
            closeRequested = true;
            if (ws != null) {
                ws.close();
            }
        };

    });
}

/**
 * Cancel order response object. Needs no translation.
 */
export interface CancelOrderResponse {
    symbol: string;
    origClientOrder: string;
    orderId: number;
    clientOrderId: string;
}

export interface BuyOrderResponse {
    trade_id: string;
}

export interface BinanceStreamBalanceResponse {
    a: string; // Asset.
    f: string; // Free amount.
    l: string; // Locked amount.
}

export interface BinanceRestBalanceResponse {
    asset: string;
    free: string;
    locked: string;
}

export class BinanceBalance {

    asset: string;

    free: number;

    locked: number;

    static fromRest(raw: BinanceRestBalanceResponse): BinanceBalance {
        const balance = new BinanceBalance();
        balance.asset = raw.asset;
        balance.free = +raw.free;
        balance.locked = +raw.locked;
        return balance;
    }

    static fromStream(raw: BinanceStreamBalanceResponse): BinanceBalance {
        const balance = new BinanceBalance();
        balance.asset = raw.a;
        balance.free = +raw.f;
        balance.locked = +raw.l;
        return balance;
    }
}

export interface BinanceRestAccountInfoResrponse {
    balances: BinanceRestBalanceResponse[];
}

export interface BinanceStreamAccountInfoResponse {
    B: BinanceStreamBalanceResponse[];
}

export class BinanceAccountInfo {

    balances: BinanceBalance[] = null;

    static fromRest(raw: BinanceRestAccountInfoResrponse): BinanceAccountInfo {
        const accountInfo = new BinanceAccountInfo();
        accountInfo.balances = raw.balances.map((b): BinanceBalance => {
            return BinanceBalance.fromRest(b);
        });
        return accountInfo;
    }

    static fromStream(raw: BinanceStreamAccountInfoResponse): BinanceAccountInfo {
        const accountInfo = new BinanceAccountInfo();
        accountInfo.balances = raw.B.map((b): BinanceBalance => {
            return BinanceBalance.fromStream(b);
        });
        return accountInfo;
    }
}

export interface RestSymbolInfo {
    symbol: string;
    status: string;
    baseAsset: string;
    baseAssetPrecision: number;
    quoteAsset: string;
    quotePrecision: number;
    filters: any[];
}

export interface RestExchangeInfoResponse {
    symbols: RestSymbolInfo[];
}

export class SymbolInfo {
    symbol: string;
    status: string;
    baseAsset: string;
    quoteAsset: string;
    filters: any[];

    minNotional: number;
    minQuantity: number;
    stepSize: number;
    tickSize: number;

    static fromRest(rest: RestSymbolInfo): SymbolInfo {
        const info = new SymbolInfo();
        Object.assign(info, rest);
        info.init();
        return info;
    }

    private init() {
        this.minNotional = this.getMinNotional();
        this.minQuantity = this.getMinQuantity();
        this.stepSize = this.getStepSize();
        this.tickSize = this.getTickSize();
    }

    getMinNotional(): number {
        if (this.filters) {
            for (const f of this.filters) {
                if (f.minNotional) {
                    return +f.minNotional;
                }
            }
        }
        return null;
    }

    getMinQuantity(): number {
        if (this.filters) {
            for (const f of this.filters) {
                if (f.filterType && f.filterType === "LOT_SIZE") {
                    return +f.minQty;
                }
            }
        }
        return null;
    }

    getStepSize(): number {
        if (this.filters) {
            for (const f of this.filters) {
                if (f.filterType && f.filterType === "LOT_SIZE") {
                    return +f.stepSize;
                }
            }
        }
        return null;
    }

    getTickSize(): number {
        if (this.filters) {
            for (const f of this.filters) {
                if (f.filterType && f.filterType === "PRICE_FILTER") {
                    return +f.tickSize;
                }
            }
        }
        return null;
    }

}

export class ExchangeInfo {

    symbols: SymbolInfo[] = [];

    static fromRest(rest: RestExchangeInfoResponse): ExchangeInfo {
        const info = new ExchangeInfo();
        for (const symbol of rest.symbols) {
            info.symbols.push(SymbolInfo.fromRest(symbol));
        }
        return info;
    }
}

export interface StreamAggTrade {
    e: string; // Event type
    E: number; // Event time
    s: string; // Symbol
    a: number; // Aggregate trade ID
    p: string; // Price
    q: string; // Quantity
    f: number; // First trade ID
    l: number; // Last trade ID
    T: number; // Trade time
    m: boolean; // Is buyer the maker
    M: boolean; // Ignore
}

export interface AggTrade {
    symbol: string;
    price: number;
    quantity: number;
}

export function buildAggTradeFromStream(raw: StreamAggTrade): AggTrade {
    return {
        symbol: raw.s.toUpperCase(),
        price: +raw.p,
        quantity: +raw.q,
    };
}

interface StreamTicker {
    e: string; // Event type.
    s: string; // Symbol.
    c: string; // Current day close price (last price).
    b: string; // Best bid price.
    a: string; // Best ask price.
    q: string; // 24h volume in quote currency.
    P: string; // 24h price change percent.
}

export interface PriceTicker {
    symbol: string;
    price: number;
    bestBid?: number;
    bestAsk?: number;
    volume?: number;
    percentChange24?: number;
}

export function buildTickerFromStream(raw: StreamTicker): PriceTicker {
    return {
        symbol: raw.s.toUpperCase(),
        price: +raw.c,
        bestBid: +raw.b,
        bestAsk: +raw.a,
        volume: +raw.q,
        percentChange24: +raw.P,
    };
}
