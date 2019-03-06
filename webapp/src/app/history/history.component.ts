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

import {AfterViewInit, Component, OnInit} from '@angular/core';
import {TradeState} from '../maker.service';
import {AppTradeState, toAppTradeState} from '../trade-table/trade-table.component';
import {MakerApiService} from "../maker-api.service";

@Component({
    selector: 'app-history',
    templateUrl: './history.component.html',
    styleUrls: ['./history.component.scss']
})
export class HistoryComponent implements OnInit, AfterViewInit {

    trades: TradeState[] = [];

    constructor(private makerApi: MakerApiService) {
    }

    ngOnInit() {
    }

    ngAfterViewInit() {
        this.makerApi.get("/api/trade/query")
            .subscribe((trades: TradeState[]) => {
                this.trades = trades.map((trade) => {
                    return toAppTradeState(trade);
                }).sort(this.sort);
            });
    }

    private sort(a: AppTradeState, b: AppTradeState) {
        return new Date(b.OpenTime).getTime() -
            new Date(a.OpenTime).getTime();
    }
}
