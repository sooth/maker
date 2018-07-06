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
import {HttpClient, HttpHeaders, HttpParams} from "@angular/common/http";
import * as hmacSHA256 from "crypto-js/hmac-sha256";
import * as hex from "crypto-js/enc-hex";
import {catchError, map} from "rxjs/operators";
import {Observable} from "rxjs";
import {throwError} from "rxjs/internal/observable/throwError";
import {OpenTradeOptions} from "./binance.service";
import {Observer} from "rxjs/Observer";

const API_ROOT = "/proxy/binance";
const STREAM_ROOT = "wss://stream.binance.com:9443";

@Injectable()
export class BinanceApiService {

    private _apiKey: string = null;

    private _apiSecret: string = null;

    constructor(private http: HttpClient) {
    }

    set apiKey(key: string) {
        this._apiKey = key;
    }

    set apiSecret(secret: string) {
        this._apiSecret = secret;
    }

    private get(path: string, params: HttpParams = null): Observable<Object> {
        const url = `${API_ROOT}${path}`;

        if (params == null) {
            params = new HttpParams();
        }

        return this.http.get<Object>(url, {
            params: params,
        });
    }

    private authenticateGet(path: string, params: HttpParams = null): Observable<Object> {
        const url = `${API_ROOT}${path}`;

        if (params == null) {
            params = new HttpParams();
        }

        let headers = new HttpHeaders();

        const timestamp = new Date().getTime();
        params = params.set("timestamp", `${timestamp}`);

        const hmacDigest = hmacSHA256(params.toString(), this._apiSecret);
        params = params.set("signature", hex.stringify(hmacDigest));

        headers = headers.append("X-MBX-APIKEY", this._apiKey);

        return this.http.get<Object>(url, {
            headers: headers,
            params: params,
        });
    }

    private post(path: string, options?: {
        params?: HttpParams;
        headers?: HttpHeaders;
    }, body: any = null) {
        const headers = options && options.headers || new HttpHeaders();
        const params = options && options.params || new HttpParams();
        return this.http.post(path, body, {
            params: params,
            headers: headers,
        }).pipe(catchError((error) => {
            if (error.error instanceof ErrorEvent) {
                console.log("A client side error occurred: ");
                console.log(error);
                return throwError(error);
            } else {
                return throwError(error);
            }
        }));
    }

    private delete(path: string, params: HttpParams = null): Observable<any> {
        return this.http.delete(path, {
            params: params,
        });
    }

    getAccountInfo(): Observable<AccountInfo> {
        const endpoint = "/api/v3/account";
        return this.authenticateGet(endpoint, null)
                .pipe(map((raw: RawRestAccountInfo) => {
                    return AccountInfo.fromRest(raw);
                }));
    }

    getExchangeInfo(): Observable<ExchangeInfo> {
        const endpoint = "/api/v1/exchangeInfo";
        return this.get(endpoint, null).pipe(map((info: RestExchangeInfoResponse) => {
            return ExchangeInfo.fromRest(info);
        }));
    }

    getPriceTicker(symbol: string): Observable<PriceTicker> {
        const endpoint = "/api/v3/ticker/price";
        const params = new HttpParams().set("symbol", symbol);
        return this.get(endpoint, params).pipe(
                map((r: RestTickerPriceResponse) => {
                    return buildTickerFromRest(r);
                }));
    }

    getBookTicker(symbol: string): Observable<BookTicker> {
        const endpoint = "/api/v3/ticker/bookTicker";
        const params = new HttpParams().set("symbol", symbol);
        return this.get(endpoint, params).pipe(
                map((r: RestBookTicker): BookTicker => {
                    return {
                        symbol: r.symbol,
                        bidPrice: +r.bidPrice,
                        bidQty: +r.bidQty,
                        askPrice: +r.askPrice,
                        askQty: +r.askQty,
                    };
                })
        );
    }

    postBuyOrder(body: OpenTradeOptions = null): Observable<BuyOrderResponse> {
        const endpoint = "/api/binance/buy";
        return <Observable<BuyOrderResponse>>this.post(endpoint, null, body);
    }

    cancelSellOrder(tradeId: string): Observable<CancelOrderResponse> {
        const endpoint = "/api/binance/sell";
        const params = new HttpParams().set("trade_id", tradeId);
        return this.delete(endpoint, params);
    }

    cancelBuy(tradeId: string): Observable<CancelOrderResponse> {
        const endpoint = "/api/binance/buy";
        const params = new HttpParams()
                .set("trade_id", tradeId);
        return this.delete(endpoint, params);
    }

    openStream(path: string): Observable<any> {
        const url = `${STREAM_ROOT}${path}`;
        return makeWebSocketObservable(url);
    }

}

interface RestTickerPriceResponse {
    symbol: string;
    price: string;
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

export interface StreamBalance {
    a: string; // Asset.
    f: string; // Free amount.
    l: string; // Locked amount.
}

export interface RestBalance {
    asset: string;
    free: string;
    locked: string;
}

export class Balance {

    asset: string;

    free: number;

    locked: number;

    static fromRest(raw: RestBalance): Balance {
        const balance = new Balance();
        balance.asset = raw.asset;
        balance.free = +raw.free;
        balance.locked = +raw.locked;
        return balance;
    }

    static fromStream(raw: StreamBalance): Balance {
        const balance = new Balance();
        balance.asset = raw.a;
        balance.free = +raw.f;
        balance.locked = +raw.l;
        return balance;
    }
}

export interface RawRestAccountInfo {
    makeCommission: number;
    takerCommission: number;
    buyerCommission: number;
    sellerCommission: number;
    canTrade: boolean;
    canWithdraw: boolean;
    canDeposit: boolean;
    updateTime: number;
    balances: RestBalance[];
}

export interface RawStreamAccountInfo {
    B: StreamBalance[];
}

export class AccountInfo {

    balances: Balance[] = null;

    static fromRest(raw: RawRestAccountInfo): AccountInfo {
        const accountInfo = new AccountInfo();
        accountInfo.balances = raw.balances.map((b): Balance => {
            return Balance.fromRest(b);
        });
        return accountInfo;
    }

    static fromStream(raw: RawStreamAccountInfo): AccountInfo {
        const accountInfo = new AccountInfo();
        accountInfo.balances = raw.B.map((b): Balance => {
            return Balance.fromStream(b);
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

interface RestExchangeInfoResponse {
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

interface MultiStreamFrame {
    stream: string;
    data: any;
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

interface StreamTicker {
    e: string; // Event type.
    s: string; // Symbol.
    c: string; // Current day close price (last price).
}

export interface PriceTicker {
    symbol: string;
    price: number;
}

export interface RestBookTicker {
    symbol: string;
    bidPrice: string;
    bidQty: string;
    askPrice: string;
    askQty: string;
}

export interface BookTicker {
    symbol: string;
    bidPrice: number;
    bidQty: number;
    askPrice: number;
    askQty: number;
}

interface StreamDepth {
    lastUpdateId: number;
    bids: any[];
    asks: any[];
}

export interface Depth {
    symbol: string;
    lastUpdateId: number;
    bids: { price: number, quantity: number }[];
    asks: { price: number, quantity: number }[];
}

export function makeDepthFromStream(symbol: string, raw: StreamDepth): Depth {
    const bids = raw.bids.map((bid) => {
        return {
            price: +bid[0],
            quantity: +bid[1],
        };
    });
    const asks = raw.asks.map((ask) => {
        return {
            price: +ask[0],
            quantity: +ask[1],
        };
    });
    return {
        symbol: symbol.toUpperCase(),
        lastUpdateId: raw.lastUpdateId,
        bids: bids,
        asks: asks,
    };
}

export function buildAggTradeFromStream(raw: StreamAggTrade): AggTrade {
    return {
        symbol: raw.s.toUpperCase(),
        price: +raw.p,
        quantity: +raw.q,
    };
}

export function buildTickerFromStream(raw: StreamTicker): PriceTicker {
    return {
        symbol: raw.s.toUpperCase(),
        price: +raw.c,
    };
}

function buildTickerFromRest(r: RestTickerPriceResponse): PriceTicker {
    return {
        symbol: r.symbol.toUpperCase(),
        price: +r.price,
    };
}

export function makeWebSocketObservable(url: string): Observable<any> {
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
