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

import {Component, OnInit} from "@angular/core";
import {HttpClient, HttpParams} from "@angular/common/http";
import {FormBuilder, FormGroup, Validators} from "@angular/forms";
import {MakerApiService} from "../maker-api.service";
import {ToastrService} from "../toastr.service";
import {Router} from "@angular/router";
import {Config, ConfigKey, ConfigService, DEFAULT_BALANCE_PERCENTS} from "../config.service";

@Component({
    selector: "app-config",
    templateUrl: "./config.component.html",
    styleUrls: ["./config.component.scss"]
})
export class ConfigComponent implements OnInit {

    config: Config = <Config>{};

    form: FormGroup = null;

    preferenceForm: FormGroup = null;

    testOk = false;

    constructor(private http: HttpClient,
                private makerApi: MakerApiService,
                private toastr: ToastrService,
                private router: Router,
                private configService: ConfigService,
                private fb: FormBuilder) {
    }

    ngOnInit() {
        this.configService.loadConfig().subscribe(() => {
            this.config = this.configService.config;
            this.buildBinanceForm();
            this.buildPreferenceForm();
        });
    }

    private buildBinanceForm() {
        this.form = this.fb.group({
            binanceApiKey: [this.config["binance.api.key"],
                Validators.required],
            binanceApiSecret: [this.config["binance.api.secret"],
                Validators.required],
        });
    }

    private buildPreferenceForm() {
        let percents: string = null;
        if (this.configService.config.balancePercents) {
            percents = this.configService.config.balancePercents;
        } else {
            percents = DEFAULT_BALANCE_PERCENTS;
        }
        this.preferenceForm = this.fb.group({
            balancePercents: [percents, Validators.pattern(/^[\d\., ]+$/)],
        });
    }

    resetBinanceForm() {
        this.buildBinanceForm();
    }

    binanceTest() {
        const formModel = this.form.value;

        const params = new HttpParams()
                .set("binance.api.key", formModel.binanceApiKey)
                .set("binance.api.secret", formModel.binanceApiSecret);
        this.http.get<any>("/api/binance/account/test", {
            params: params,
        }).subscribe((response) => {
            if (!response.ok) {
                this.toastr.error("Authentication test failed.");
                this.testOk = false;
            } else {
                this.toastr.info("Authentication OK.");
                this.testOk = true;
            }
        });
    }

    binanceSave() {
        const formModel: BinanceForm = <BinanceForm>this.form.value;
        this.configService.set(ConfigKey.BINANCE_API_KEY, formModel.binanceApiKey);
        this.configService.set(ConfigKey.BINANCE_API_SECRET, formModel.binanceApiSecret);
        this.configService.saveBinanceConfig().subscribe(() => {
            (<any>window).location = "/";
        }, (error) => {
            this.toastr.error(`Failed to save configuration: ${JSON.stringify(error)}.`);
        });
    }

    savePreferences() {
        const prefs: PreferenceForm = <PreferenceForm>this.preferenceForm.value;
        this.configService.set(ConfigKey.PREFERENCE_BALANCE_PERCENTS, prefs.balancePercents);
        this.configService.config.balancePercents = prefs.balancePercents;
        this.configService.savePreferences().subscribe(() => {
            this.toastr.success("Preferences saved.");
        }, (error) => {
            this.toastr.error(`Failed to save preferences: ${JSON.stringify(error)}`);
        });
    }

    resetPreferences() {
        this.buildPreferenceForm();
    }

}

interface BinanceForm {
    binanceApiKey: string;
    binanceApiSecret: string;
}

interface PreferenceForm {
    balancePercents: string;
}
