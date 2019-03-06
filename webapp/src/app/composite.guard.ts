// Copyright (C) 2019 Cranky Kernel
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
import {ActivatedRouteSnapshot, CanActivate, Router, RouterStateSnapshot} from '@angular/router';
import {Observable, of} from 'rxjs';
import {LoginService} from "./login.service";
import {map, switchMap, tap} from "rxjs/operators";
import {MakerApiService} from "./maker-api.service";
import {ToastrService} from "./toastr.service";

@Injectable({
    providedIn: 'root'
})
export class CompositeGuard implements CanActivate {

    constructor(private loginService: LoginService,
                private makerApi: MakerApiService,
                private toastr: ToastrService,
                private router: Router) {
    }

    canActivate(
        next: ActivatedRouteSnapshot,
        state: RouterStateSnapshot): Observable<boolean> | Promise<boolean> | boolean {
        return this.loginService.checkLogin()
            .pipe(tap((ok) => {
                if (!ok) {
                    this.router.navigate(["/login"]);
                }
            }), switchMap((loginOk) => {
                if (loginOk && next.data.configRequired === true) {
                    console.log("Login is OK, checking configuration.");
                    return this.makerApi.getConfig()
                        .pipe(map((config) => {
                            if (!(config["binance.api.key"] && config["binance.api.secret"])) {
                                this.toastr.error("Incomplete Binance configuration. Redirecting to configuration page.");
                                this.router.navigate(["/config"]);
                                return false;
                            }
                            return true;
                        }))
                } else {
                    return of(loginOk);
                }
            }));
    }

}
