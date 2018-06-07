import {Injectable} from '@angular/core';
import {HttpClient, HttpHeaders, HttpParams} from '@angular/common/http';
import * as hmacSHA256 from 'crypto-js/hmac-sha256';
import * as hex from 'crypto-js/enc-hex';
import * as moment from "moment";
import {map} from 'rxjs/operators';
import {Observable} from 'rxjs';

declare var localStorage: any;

const API_ROOT = "https://api.quadrigacx.com";

export enum OrderType {
    BUY = 0,
    SELL = 1,
}

export interface Order {
    amount: number;
    price: number;
    book: string;
}

export interface ActiveOrder {
    id: string;
    amount: number;
    price: number;
    type: OrderType;
    book: string;
    datetime: moment.Moment;
    _source: any;
}

export interface OrderBookResponseItem {
    price: number;
    volume: number;
}

export interface OrderBookResponse {
    timestamp: moment.Moment;
    asks: OrderBookResponseItem[];
    bids: OrderBookResponseItem[];
}

export interface WsOrderEntry {
    // Looks like a timestamp, but also an order ID.
    d: number;

    // Order type: 0 = Buy, 1 = Sell.
    t: number;

    // Amount, volume.
    a?: number;

    // Price.
    r: number;

    // Value. If no value, the order has been deleted.
    v?: number;
}

export interface WsStats {
    // High.
    h: string;

    // Low
    l: string;

    // Volume
    v: string;

    // Last trade price.
    t: string;
}

export interface WsUpdateGlobal {
    book: string;
    orders: WsOrderEntry[];
    trades: any[];
    hash: number;
    sequence: number;
    stats: WsStats;
}

export const QuadrigaBook = {
    BTCCAD: "btc_cad",
    BTCUSD: "btc_usd",
    ETHCAD: "eth_cad",
    ETHBTC: "eth_btc",
    LTCCAD: "ltc_cad",
    LTCBTC: "ltc_btc",
    BCHCAD: "bch_cad",
    BCHBTC: "bch_btc",
};

@Injectable()
export class QuadrigacxService {

    public books: string[] = [
        "btc_cad",
        "btc_usd",
        "eth_cad",
        "eth_btc",
        "ltc_cad",
        "ltc_btc",
        "bch_cad",
        "bch_btc",
        //"btg_cad",
        //"btg_btc",
    ];

    private previousNonce: number = 0;

    public defaultBook: string = this.books[0];

    constructor(private http: HttpClient) {
    }

    get(endpoint, options = {}) {
        let url = `${API_ROOT}/${endpoint}`;
        return this.http.get(url, options);
    }

    getTicker(book: string = null) {
        let endpoint = `/v2/ticker`;
        let params = new HttpParams();
        if (book) {
            params = params.append("book", book);
        }
        return this.get(endpoint, {params: params});
    }

    getOpenOrders(book: string): Observable<any> {
        let endpoint = "/v2/open_orders";
        let params = this.authenticateParams(new HttpParams());
        params = params.append("book", book);
        return this.post(endpoint, params);
    }

    getBalance() {
        let endpoint = `/v2/balance`;

        let creds = this.getCredentials();
        let nonce = this.getNonce();
        let msg = `${nonce}${creds.clientId}${creds.apiKey}`;
        let hmacDigest = hmacSHA256(msg, creds.apiSecret);
        let signature = hex.stringify(hmacDigest);

        let params = new HttpParams();
        params = params.append("key", creds.apiKey);
        params = params.append("nonce", `${nonce}`);
        params = params.append("signature", signature);

        let headers = new HttpHeaders({
            'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
        });

        return this.http.post(`${API_ROOT}${endpoint}`, params, {
            headers: headers,
        });
    }

    post(endpoint: string, params: HttpParams) {
        let headers = new HttpHeaders({
            'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
        });
        return this.http.post(`${API_ROOT}${endpoint}`, params, {
            headers: headers,
        });
    }

    getEngineOrders(book: string, t: number = null): Observable<any> {
        let url = `https://www.quadrigacx.com/engine/orders/${book}`;
//        let url = `http://localhost:6035/api/1/quadriga/engine/orders/${book}`;
        let params = new HttpParams();
        if (t != null) {
            params = params.append("t", t.toString());
        }
        return this.http.get(url, {params: params});
    }

    getOrderBook(book: string): Observable<OrderBookResponse> {
        let params = new HttpParams().append("book", book);
        return this.get("/v2/order_book", {params: params})
                .pipe(map((response: any) => {
                    return {
                        timestamp: moment(response.timestamp, "X"),
                        asks: response.asks.map((x) => {
                            return {
                                price: x[0],
                                volume: x[1],
                            }
                        }),
                        bids: response.bids.map((x) => {
                            return {
                                price: x[0],
                                volume: x[1],
                            }
                        }),
                    }
                }));
    }

    buy(order: Order): Observable<ActiveOrder> {
        let params = new HttpParams();
        params = params.append("amount", order.amount.toString());
        params = params.append("price", order.price.toString());
        params = params.append("book", order.book);
        params = this.authenticateParams(params);
        return this.post("/v2/buy", params)
                .pipe(map((response: any) => {
                    let datetime = moment(
                            response.datetime, "YYYY-MM-DD HH:mm:ss");
                    let activeOrder = {
                        id: response.id,
                        amount: +response.amount,
                        price: +response.price,
                        type: +response.type,
                        book: response.book,
                        datetime: datetime,
                        _source: response,
                    };
                    return activeOrder;
                }))
    }

    sell(order: Order): Observable<ActiveOrder> {
        let params = new HttpParams();
        params = params.append("amount", order.amount.toString());
        params = params.append("price", order.price.toString());
        params = params.append("book", order.book);
        params = this.authenticateParams(params);
        return this.post("/v2/sell", params)
                .pipe(map((response: any) => {
                    let datetime = moment(
                            response.datetime, "YYYY-MM-DD HH:mm:ss");
                    let activeOrder = {
                        id: response.id,
                        amount: +response.amount,
                        price: +response.price,
                        type: +response.type,
                        book: response.book,
                        datetime: datetime,
                        _source: response,
                    };
                    return activeOrder;
                }))
    }

    cancel(orderId: string) {
        let params = new HttpParams();
        params = params.append("id", orderId);
        params = this.authenticateParams(params);
        return this.post("/v2/cancel_order", params);
    }

    private getNonce(): number {
        let nonce = new Date().getTime();
        while (nonce <= this.previousNonce) {
            console.log("incrementing nonce");
            nonce = nonce + 1;
        }
        this.previousNonce = nonce;
        return nonce;
    }

    connect(): Observable<any> {
        let params = new HttpParams();
        params = params.append("transport", "websocket");
        let ws = new WebSocket(`wss://realtime.quadrigacx.com/?${params.toString()}`);

        return Observable.create((observer) => {

            let interval = setInterval(() => {
                switch (ws.readyState) {
                    case ws.CONNECTING:
                        // Do nothing.
                        break;
                    case ws.OPEN:
                        // Send keepalive.
                        ws.send("2");
                        break;
                    default:
                        clearInterval(interval);
                        break;
                }
            }, 20000);

            ws.onmessage = (event) => {
                let parts = event.data.match(/(\d+)(.*)/);
                let prefix = parts[1];

                let body = null;
                if (parts[2].length > 0) {
                    body = JSON.parse(parts[2]);
                }

                if (prefix == 0) {
                    // A control message. Log but don't pass on.
                    console.log(body);
                    return;
                }

                observer.next({
                    prefix: parts[1],
                    body: body,
                });
            };

            ws.onclose = () => {
                observer.complete();
            };

            ws.onerror = (event) => {
                observer.error(event);
            };

            return () => {
                ws.close();
            }
        })
    }

    private getCredentials(): any {
        let clientId = localStorage.CLIENT_ID;
        let apiKey = localStorage.API_KEY;
        let apiSecret = localStorage.API_SECRET;
        return {
            clientId: clientId,
            apiKey: apiKey,
            apiSecret: apiSecret,
        };
    }

    private authenticateParams(params: HttpParams): HttpParams {
        let creds = this.getCredentials();
        let nonce = this.getNonce();
        let msg = `${nonce}${creds.clientId}${creds.apiKey}`;
        let hmacDigest = hmacSHA256(msg, creds.apiSecret);
        let signature = hex.stringify(hmacDigest);
        params = params.append("key", creds.apiKey);
        params = params.append("nonce", `${nonce}`);
        params = params.append("signature", signature);
        return params;
    }
}
