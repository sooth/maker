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

import {Injectable} from '@angular/core';
import {Observable, of, ReplaySubject} from "rxjs";
import {catchError, map, tap} from "rxjs/operators";
import {MakerApiService} from "./maker-api.service";
import {Router} from "@angular/router";

declare var localStorage: any;

@Injectable({
    providedIn: 'root'
})
export class LoginService {

    $onLogin = new ReplaySubject<boolean>(1);

    authenticated = false;

    constructor(private makerApi: MakerApiService,
                private router: Router) {
    }

    private setAuthenticated() {
        if (!this.authenticated) {
            this.authenticated = true;
            this.$onLogin.next(true);
        }
    }

    checkLogin(): Observable<boolean> {
        if (this.authenticated) {
            return of(true);
        }

        const sessionId = localStorage.getItem("sessionId");
        if (sessionId !== null) {
            this.makerApi.setSessionId(sessionId);
        }

        return this.makerApi.get("/api/version")
            .pipe(map((response: any) => {
                this.setAuthenticated();
                return true;
            }), catchError((err) => {
                return of(false);
            }));
    }

    login(username: string, password: string): Observable<any> {
        return this.makerApi.login(username, password)
            .pipe(tap((response: any) => {
                localStorage.setItem("sessionId", response.sessionId);
                this.makerApi.setSessionId(response.sessionId);
                this.setAuthenticated();
            }));
    }

    private clearSessionId() {
        localStorage.removeItem("sessionId");
        this.makerApi.setSessionId(null);
    }

    gotoLogin() {
        this.router.navigate(["/login"])
            .then(() => {
                location.reload(true);
            });
    }

    logout() {
        this.clearSessionId();
        window.location.reload(true);
    }
}
