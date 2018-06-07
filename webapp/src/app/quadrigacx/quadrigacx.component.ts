import {Component, OnDestroy, OnInit} from '@angular/core';
import {
    OrderType,
    QuadrigaBook,
    QuadrigacxService,
    WsOrderEntry,
    WsStats,
    WsUpdateGlobal
} from './quadrigacx.service';
import {ActivatedRoute, Router} from '@angular/router';

import {Subscription} from 'rxjs';

declare var localStorage: any;

class Ticker {
    last: number;
    bid: number;
    ask: number;
    spread: number;
    spreadPercent: number;
}

class OrderBook {
    ticker: Ticker = new Ticker();
    bidsMap: any = {};
    bidsArray: any[] = [];
    asksMap: any = {};
    asksArray: any[] = [];
    buyPct?: number = null;

    constructor(public name: string) {
    }
}

@Component({
    selector: 'app-quadrigacx',
    templateUrl: './quadrigacx.component.html',
    styleUrls: ['./quadrigacx.component.scss']
})
export class QuadrigacxComponent implements OnInit, OnDestroy {

    private defaultBook: string = "ltc_cad";

    books: string[];
    book: string;

    model = {
        buy: {
            amount: null,
            price: null,
        },
        sell: {
            amount: null,
            price: null,
        },
        myOrder: {
            bid: null,
            ask: null,
            spread: null,
        }
    };

    order = {
        book: "",
        buy: {
            amount: null,
            price: null,
        },
        sell: {
            amount: null,
            price: null,
        },
    };

    orderBooks = {};

    private initState: string = JSON.stringify({
        model: this.model,
        //orders: this.orders,
    });

    private websocket$: Subscription = null;

    constructor(private quadrigacx: QuadrigacxService,
                private router: Router,
                private route: ActivatedRoute) {
        this.books = quadrigacx.books;
        for (let book of this.books) {
            this.orderBooks[book] = new OrderBook(book);
        }
    }

    ngOnInit() {
        this.route.params.subscribe((params) => {
            this.book = params.book || this.defaultBook;
            this.init();
        });
    }

    ngOnDestroy() {
        this.websocket$.unsubscribe();
    }

    private init() {
        let initState = JSON.parse(this.initState);
        //this.orders = initState.orders;
        this.model = initState.model;

        for (const book in this.orderBooks) {
            this.quadrigacx.getEngineOrders(book).subscribe((response) => {
                response.bids.forEach((e) => {
                    this.orderBooks[book].bidsMap[e.d] = {
                        price: e.r,
                        amount: e.a,
                        value: e.v,
                    }
                });
                response.asks.forEach((e) => {
                    this.orderBooks[book].asksMap[e.d] = {
                        price: e.r,
                        amount: e.a,
                        value: e.v,
                    }
                });
                this.sortOrders(this.orderBooks[book]);
            });

            // Commented out as it eats up the API limit.
            // this.quadrigacx.getTicker(book).subscribe((ticker: any) => {
            //     this.orderBooks[book].ticker.last = +ticker.last;
            // })
        }

        if (this.websocket$ != null) {
            return;
        }

        // Just websocket below.
        this.websocket$ = this.quadrigacx.connect().subscribe((event) => {
            if (event.body) {
                const command = event.body[0];

                switch (command) {
                    case "update-global":
                        break;
                    default:
                        console.log("unknown websocket command: " + command);
                        console.log(event);
                        break;
                }

                let body = <WsUpdateGlobal>event.body[1];

                if (!body) {
                    return;
                }

                let book = body.book;

                if (body.trades) {
                    console.log(body);
                    for (let trade of body.trades) {
                        if (trade.t == OrderType.BUY) {
                            console.log(`BUY: ${book} - ${JSON.stringify(trade)}`);
                        } else if (trade.t == OrderType.SELL) {
                            console.log(`SELL: ${book} - ${JSON.stringify(trade)}`);
                        }
                    }
                }

                let orders = <WsOrderEntry[]>body.orders;

                if (book in this.orderBooks && body.stats) {
                    const stats = <WsStats>body.stats;
                    this.orderBooks[book].ticker.last = +stats.t;
                }

                if (book in this.orderBooks && orders) {
                    for (let order of orders) {
                        if (order.t == OrderType.BUY) {
                            if (!order.v) {
                                delete(this.orderBooks[book].bidsMap[order.d]);
                            } else {
                                this.orderBooks[book].bidsMap[order.d] = {
                                    price: order.r,
                                    amount: order.a,
                                    value: order.v,
                                }
                            }
                        } else if (order.t == OrderType.SELL) {
                            if (!order.v) {
                                delete(this.orderBooks[book].asksMap[order.d]);
                            } else {
                                let ask = {
                                    id: order.d,
                                    price: order.r,
                                    amount: order.a,
                                    value: order.v,
                                };
                                this.orderBooks[book].asksMap[order.d] = ask;
                            }
                        }
                    }
                    this.sortOrders(this.orderBooks[book]);

                    // Calculate the percentage of order volume that is buy.
                    let orderBook = this.orderBooks[book];
                    let totalAskVolume = 0;
                    let totalBidVolume = 0;
                    for (let ask in orderBook.asksMap) {
                        let order = orderBook.asksMap[ask];
                        totalAskVolume += order.amount;
                    }
                    for (let id in orderBook.bidsMap) {
                        let order = orderBook.bidsMap[id];
                        totalBidVolume += order.amount;
                    }
                    orderBook.buyPct = (totalBidVolume / (totalBidVolume + totalAskVolume));
                }

                this.arb["LTC_BTC"] = {
                    cashIn: 100,
                    cashOut: ((100 / this.orderBooks[QuadrigaBook.LTCCAD].ticker.ask * 0.995) *
                            (this.orderBooks[QuadrigaBook.LTCBTC].ticker.bid * 0.995)) *
                    (this.orderBooks[QuadrigaBook.BTCCAD].ticker.bid * 0.995),
                };

                this.arb["BTC_LTC"] = {
                    cashIn: 100,
                    cashOut: ((100 / this.orderBooks[QuadrigaBook.BTCCAD].ticker.ask * 0.995) /
                            (this.orderBooks[QuadrigaBook.LTCBTC].ticker.ask * 0.995)) *
                    (this.orderBooks[QuadrigaBook.LTCCAD].ticker.bid * 0.995),
                };

                this.arb["ETH_BTC"] = {
                    cashIn: 100,
                    cashOut: ((100 / this.orderBooks[QuadrigaBook.ETHCAD].ticker.ask * 0.995) *
                            (this.orderBooks[QuadrigaBook.ETHBTC].ticker.bid * 0.995)) *
                    (this.orderBooks[QuadrigaBook.BTCCAD].ticker.bid * 0.995),
                };

                this.arb["BTC_ETH"] = {
                    cashIn: 100,
                    cashOut: ((100 / this.orderBooks[QuadrigaBook.BTCCAD].ticker.ask * 0.995) /
                            (this.orderBooks[QuadrigaBook.ETHBTC].ticker.ask * 0.995)) *
                    (this.orderBooks[QuadrigaBook.ETHCAD].ticker.bid * 0.995),
                };

                this.arb["BCH_BTC"] = {
                    cashIn: 100,
                    cashOut: ((100 / this.orderBooks[QuadrigaBook.BCHCAD].ticker.ask * 0.995) *
                            (this.orderBooks[QuadrigaBook.BCHBTC].ticker.bid * 0.995)) *
                    (this.orderBooks[QuadrigaBook.BTCCAD].ticker.bid * 0.995),
                };

                this.arb["BTC_BCH"] = {
                    cashIn: 100,
                    cashOut: ((100 / this.orderBooks[QuadrigaBook.BTCCAD].ticker.ask * 0.995) /
                            (this.orderBooks[QuadrigaBook.BCHBTC].ticker.ask * 0.995)) *
                    (this.orderBooks[QuadrigaBook.BCHCAD].ticker.bid * 0.995),
                };

            }
        });
    }

    arb: any = {};

    sortOrders(book) {
        book.bidsArray = Object.keys(book.bidsMap).map((orderId) => {
            return book.bidsMap[orderId];
        }).sort((a, b) => {
            return b.price - a.price;
        }).slice(0, 50);

        book.asksArray = Object.keys(book.asksMap).map((orderId) => {
            return book.asksMap[orderId];
        }).sort((a, b) => {
            return a.price - b.price;
        }).slice(0, 50);

        if (book.bidsArray.length > 0 && book.asksArray.length > 0) {
            book.ticker.bid = book.bidsArray[0].price;
            book.ticker.ask = book.asksArray[0].price;
            this.calculateSpread(book);
        }
    }

    calculateSpread(book) {
        book.ticker.spread = book.ticker.ask - book.ticker.bid;
        book.ticker.spreadPercent = book.ticker.spread / book.ticker.bid;
    }

    private round(val: number, x: number = 100000000): number {
        return Math.round(val * x) / x;
    }

    submitBuy() {
        this.quadrigacx.buy({
            book: this.book,
            amount: this.model.buy.amount,
            price: this.model.buy.price,
        }).subscribe((response) => {
            console.log(response);
            // this.quadrigacx.cancel(response.id).subscribe(response => {
            //     console.log(response);
            // });
        })
    }

    submitSell() {
        this.quadrigacx.sell({
            book: this.book,
            amount: this.model.sell.amount,
            price: this.model.sell.price,
        }).subscribe((response) => {
            console.log(response);
            // this.quadrigacx.cancel(response.id).subscribe(response => {
            //     console.log(response);
            // });
        })
    }

    submitBuySell() {
        this.submitBuy();
        setTimeout(() => {
            this.submitSell();
        }, 100);
    }

    populate() {
        let buyAmount = 0.3;
        let sellAmount = buyAmount - (buyAmount * 0.005);

        this.model.buy.amount = buyAmount;
        this.model.sell.amount = sellAmount;

        if (this.book.endsWith("btc")) {
            this.model.buy.price = this.round(this.model.myOrder.bid, 100000000);
            this.model.sell.price = this.round(this.model.myOrder.ask, 100000000);
        } else {
            this.model.buy.price = this.round(this.model.myOrder.bid, 100);
            this.model.sell.price = this.round(this.model.myOrder.ask, 100);
        }
    }

    changeBook() {
        this.router.navigate(['/quadrigacx', {book: this.book}]);
    }
}
