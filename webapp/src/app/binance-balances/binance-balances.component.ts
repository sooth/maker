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

import {Component, OnInit} from '@angular/core';
import {BinanceAccountInfo, BinanceBalance} from "../binance-api.service";
import {BinanceProxyService} from "../binance-proxy.service";

@Component({
    selector: 'app-binance-balances',
    templateUrl: './binance-balances.component.html',
    styleUrls: ['./binance-balances.component.scss']
})
export class BinanceBalancesComponent implements OnInit {

    accountInfo: BinanceAccountInfo = null;

    balances: BalanceEntry[] = [];

    constructor(private binanceProxy: BinanceProxyService) {
    }

    ngOnInit() {
        this.refresh();
    }

    refresh() {
        this.binanceProxy.getAccountInfo().subscribe((accountInfo) => {
            this.accountInfo = accountInfo;
            this.balances = accountInfo.balances
                .map((binanceBalance) => {
                    let balance: BalanceEntry = <BalanceEntry>binanceBalance;
                    balance.total = balance.free + balance.locked;
                    return balance;
                })
                .filter((balance) => {
                    return balance.total > 0;
                })
                .sort((a, b) => {
                    return a.total - b.total;
                })
        });
    }

}

interface BalanceEntry extends BinanceBalance {
    total: number;
}