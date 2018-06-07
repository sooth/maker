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
import {MakerApiService} from "./maker-api.service";
import {Observable, of} from "rxjs";
import {map, tap} from "rxjs/operators";
import {HttpClient} from "@angular/common/http";

export const DEFAULT_BALANCE_PERCENTS = "5,10,25,50,75,100";

@Injectable({
    providedIn: "root"
})
export class ConfigService {

    config: Config = null;

    constructor(private makerApi: MakerApiService,
                private http: HttpClient) {
    }

    loadConfig(): Observable<any> {
        if (this.config) {
            return of(this.config);
        }
        return this.makerApi.getConfig().pipe(tap((config) => {
            this.config = config;
        }));
    }

    getBalancePercents(): number[] {
        let percents = DEFAULT_BALANCE_PERCENTS;
        if (this.config[ConfigKey.PREFERENCE_BALANCE_PERCENTS]) {
            percents = this.config[ConfigKey.PREFERENCE_BALANCE_PERCENTS];
        }
        return percents.split(",").map((val) => {
            return +val;
        }).sort((a, b) => {
            return a - b;
        });
    }

    set(key: string, val: any) {
        this.config[key] = val;
    }

    saveBinanceConfig(): Observable<boolean> {
        return this.http.post("/api/binance/config", {
            key: this.config[ConfigKey.BINANCE_API_KEY],
            secret: this.config[ConfigKey.BINANCE_API_SECRET],
        }).pipe(map(() => {
            return true;
        }));
    }

    savePreferences(): Observable<boolean> {
        return this.makerApi.savePreferences(this.config);
    }
}

export interface Config {
    [key: string]: any;
}

export enum ConfigKey {
    "BINANCE_API_KEY" = "binance.api.key",
    "BINANCE_API_SECRET" = "binance.api.secret",
    "PREFERENCE_BALANCE_PERCENTS" = "preferences.balance.percents",
}
