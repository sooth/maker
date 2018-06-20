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

import {Component, OnDestroy, OnInit} from '@angular/core';
import {HttpClient} from '@angular/common/http';
import {ActivatedRoute} from '@angular/router';
import {Subscription} from 'rxjs/Subscription';
import {MakerService, TradeState} from '../maker.service';
import {
    AppTradeState,
    toAppTradeState
} from '../trade-table/trade-table.component';
import {BinanceService} from '../binance.service';
import {Trade} from '../trade';

@Component({
    selector: 'app-trade-detail',
    templateUrl: './trade-detail.component.html',
    styleUrls: ['./trade-detail.component.scss']
})
export class TradeDetailComponent implements OnInit, OnDestroy {

    private subs: Subscription[] = [];

    trade: AppTradeState = null;

    constructor(private http: HttpClient,
                private route: ActivatedRoute,
                private binance: BinanceService,
                private maker: MakerService) {
    }

    ngOnInit() {
        let s = this.route.params.subscribe((params) => {
            this.http.get(`/api/trade/${params.tradeId}`).subscribe((trade: TradeState) => {
                this.trade = toAppTradeState(trade);
            })
        });
        this.subs.push(s);

        s = this.maker.trade$.subscribe((trade) => {
            if (this.trade && this.trade.LocalID == trade.LocalID) {
                this.trade = toAppTradeState(trade);
            }
        });
        this.subs.push(s);

        s = this.maker.binanceAggTrades$.subscribe((trade) => {
            if (this.trade && Trade.isOpen(this.trade) &&
                    this.trade.Symbol == trade.symbol) {
                Trade.onLastPrice(this.trade, trade.price);
            }
        });
        this.subs.push(s);

    }

    ngOnDestroy() {
        for (const sub of this.subs) {
            sub.unsubscribe();
        }
    }
}
