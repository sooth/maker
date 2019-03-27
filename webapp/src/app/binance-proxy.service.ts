// Copyright (C) 2018-2019 Cranky Kernel
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

import {Injectable} from '@angular/core';
import {HttpParams} from "@angular/common/http";
import {Observable} from "rxjs";
import {MakerApiService} from "./maker-api.service";
import {map} from "rxjs/operators";
import {
    BinanceAccountInfo,
    BinanceRestAccountInfoResrponse,
    ExchangeInfo,
    RestExchangeInfoResponse
} from "./binance-api.service";

@Injectable({
    providedIn: 'root'
})
export class BinanceProxyService {

    constructor(private makerApi: MakerApiService) {
    }

    private get(path: string, params: HttpParams = new HttpParams()): Observable<Object> {
        const url = `${path}`;
        return this.makerApi.get(url, {
            params: params,
        });
    }

    getTicker24h(symbol: string): Observable<Ticker24hResponse> {
        const endpoint = "/proxy/binance/api/v1/ticker/24hr";
        const params = new HttpParams().set("symbol", symbol);
        return <Observable<Ticker24hResponse>>this.get(endpoint, params);
    }

    getAccountInfo(): Observable<BinanceAccountInfo> {
        return this.makerApi.get("/api/binance/proxy/getAccount")
            .pipe(map((restResponse: BinanceRestAccountInfoResrponse) => {
                const accountInfo = BinanceAccountInfo.fromRest(restResponse);
                return accountInfo;
            }));
    }

    getExchangeInfo(): Observable<ExchangeInfo> {
        const endpoint = "/proxy/binance/api/v1/exchangeInfo";
        return this.get(endpoint, null).pipe(map((info: RestExchangeInfoResponse) => {
            return ExchangeInfo.fromRest(info);
        }));
    }

}

export interface Ticker24hResponse {
    symbol: string;
    priceChangePercent: string;
    lastPrice: string;
    bidPrice: string;
    askPrice: string;
    quoteVolume: string;
}
