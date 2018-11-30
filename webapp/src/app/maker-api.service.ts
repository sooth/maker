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
import {HttpClient} from "@angular/common/http";
import {Observable} from "rxjs";
import {map} from "rxjs/operators";
import {LimitSellType} from './binance.service';

/**
 * The MakerApiService contains methods that wrap around calls to the Maker
 * API.
 */
@Injectable({
    providedIn: "root"
})
export class MakerApiService {

    constructor(private http: HttpClient) {
    }

    getConfig(): Observable<any> {
        return this.http.get("/api/config")
                .pipe(map((response) => {
                    return flattenJson(response);
                }));
    }

    savePreferences(prefs: any): Observable<boolean> {
        return this.http.post("/api/config/preferences", prefs)
                .pipe(map(() => {
                    return true;
                }));
    }

}

function flattenJson(input) {
    const output = {};

    for (const i in input) {
        if (!input.hasOwnProperty(i)) {
            continue;
        }

        if ((typeof input[i]) === "object") {
            const flatObject = flattenJson(input[i]);
            for (const x in flatObject) {
                if (!flatObject.hasOwnProperty(x)) {
                    continue;
                }

                output[i + "." + x] = flatObject[x];
            }
        } else {
            output[i] = input[i];
        }
    }

    return output;
}
