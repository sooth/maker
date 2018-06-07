import {Injectable} from "@angular/core";
import {ActivatedRouteSnapshot, CanActivate, Router, RouterStateSnapshot} from "@angular/router";
import {Observable} from "rxjs";
import {MakerApiService} from "./app/maker-api.service";
import {map} from "rxjs/operators";
import {ToastrService} from "./app/toastr.service";

/**
 * The BinanceConfigGuard checks that a valid configuration exists. If it
 * doesn't, the user will be redirected to the configuraion page.
 */
@Injectable({
    providedIn: "root"
})
export class BinanceConfigGuard implements CanActivate {

    constructor(private makerApi: MakerApiService,
                private router: Router,
                private toastr: ToastrService) {
    }

    canActivate(next: ActivatedRouteSnapshot,
                state: RouterStateSnapshot): Observable<boolean> | boolean {
        return this.makerApi.getConfig().pipe(map((config) => {
            if (!(config["binance.api.key"] && config["binance.api.secret"])) {
                this.toastr.error("Incomplete Binance configuration. Redirecting to configuration page.");
                this.router.navigate(["/config"]);
                return false;
            }
            return true;
        }));
    }
}
