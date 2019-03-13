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
import {HttpClient, HttpHeaders} from "@angular/common/http";
import {Observable} from "rxjs";
import {map} from "rxjs/operators";

const SESSION_HEADER = "X-Session-ID";

/**
 * The MakerApiService contains methods that wrap around calls to the Maker
 * API.
 */
@Injectable({
    providedIn: "root"
})
export class MakerApiService {

    private sessionId: string = null;

    constructor(private http: HttpClient) {
    }

    post(url: string, body: any, options: any = null): Observable<any> {
        if (!options) {
            options = {};
        }
        let headers = options.headers || new HttpHeaders();
        if (this.sessionId) {
            headers = headers.set(SESSION_HEADER, this.sessionId);
        }
        options.headers = headers;
        return this.http.post(url, body, options);
    }

    get(url: string, options: any = null): Observable<any> {
        if (!options) {
            options = {};
        }
        let headers = options.headers || new HttpHeaders();
        if (this.sessionId) {
            headers = headers.set(SESSION_HEADER, this.sessionId);
        }
        options.headers = headers;
        return this.http.get(url, options);
    }

    delete(url: string, options: any = null): Observable<any> {
        if (!options) {
            options = {};
        }
        let headers = options.headers || new HttpHeaders();
        if (this.sessionId) {
            headers = headers.set(SESSION_HEADER, this.sessionId);
        }
        options.headers = headers;
        return this.http.delete(url, options);
    }

    setSessionId(sessionId: string) {
        console.log(`Session ID set to ${sessionId}`);
        this.sessionId = sessionId;
    }

    getConfig(): Observable<any> {
        return this.get("/api/config")
            .pipe(map((response) => {
                return flattenJson(response);
            }));
    }

    savePreferences(prefs: any): Observable<boolean> {
        return this.post("/api/config/preferences", prefs)
            .pipe(map(() => {
                return true;
            }));
    }

    login(username: string, password: string): Observable<any> {
        return this.http.post("/api/login", {
            username: username,
            password: password,
        });
    }

    getVersion(): Observable<any> {
        return this.get("/api/version");
    }

    openWebsocket(): WebSocket {
        let proto = window.location.protocol == "https:" ? "wss" : "ws";
        let url = `${proto}://${window.location.host}/ws?`;
        if (this.sessionId) {
            url = `${url}&sessionId=${this.sessionId}`;
        }
        console.log(`Opening websocket: ${url}`);
        return new WebSocket(url);
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
