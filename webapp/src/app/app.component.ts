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
import {BinanceService} from './binance.service';
import {LoginService} from "./login.service";
import {MakerApiService} from "./maker-api.service";
import {MakerSocketService, MakerSocketState} from "./maker-socket.service";
import {MakerService} from "./maker.service";

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.scss']
})
export class AppComponent implements OnInit {

    ticker: { [key: string]: number } = {};

    alertClass: string = "alert-dark";

    status: any = {
        makerSocketOk: false,
        makerSocketState: "initializing",
        binanceUserSocketOk: false,
        binanceUserSocketState: "initializing",
    };

    constructor(private binance: BinanceService,
                public makerApi: MakerApiService,
                public loginService: LoginService,
                private makerService: MakerService,
                private makerSocket: MakerSocketService) {
        this.status.makerSocketState = makerSocket.state;
    }

    ngOnInit() {
        this.binance.isReady$.subscribe(() => {
            this.binance.subscribeToTicker("BTCUSDT").subscribe((ticker) => {
                this.ticker[ticker.symbol] = ticker.price;
            });
        });

        this.makerSocket.stateChange$.subscribe((state: string) => {
            console.log("Make socket state changed: " + state);
            if (state === MakerSocketState.CONNECTED) {
                this.status.makerSocketOk = true;
                this.status.makerSocketState = "OK";
            } else {
                this.status.makerSocketOk = false;
                this.status.makerSocketState = state;

                this.status.binanceUserSocketState = "unknown";
                this.status.binanceUserSocketOk = false;
            }
            this.updateAlertColor();
        });

        this.makerService.statusUpdate$.subscribe((status) => {
            console.log("Maker server status: " + JSON.stringify(status));
            if (status.binanceUserSocketState === "ok") {
                this.status.binanceUserSocketOk = true;
                this.status.binanceUserSocketState = "OK"
            } else {
                this.status.binanceUserSocketOk = false;
                this.status.binanceUserSocketState = status.binanceUserSocketState || "unknown";
            }
            this.updateAlertColor();
        });

    }

    private updateAlertColor() {
        if (this.status.makerSocketOk && this.status.binanceUserSocketOk) {
            this.alertClass = "alert-success";
        } else {
            this.alertClass = "alert-danger";
        }
    }

    logout() {
        this.loginService.logout();
    }
}
