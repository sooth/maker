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

import {BrowserModule} from "@angular/platform-browser";
import {NgModule} from "@angular/core";

import {AppRoutingModule} from "./app-routing.module";
import {AppComponent} from "./app.component";
import {FormsModule, ReactiveFormsModule} from "@angular/forms";
import {BrowserAnimationsModule} from "@angular/platform-browser/animations";
import {BinanceService} from "./binance.service";
import {ToastrService} from "./toastr.service";
import {StopLossFormComponent} from "./stoplossform/stop-loss-form.component";
import {WithQuoteAssetPipe} from "./pipes/withquoteasset.pipe";
import {TradeTableComponent} from "./trade-table/trade-table.component";
import {BinanceApiService} from "./binance-api.service";
import {TradeComponent} from "./trade/trade.component";
import {ConfigComponent} from "./config/config.component";

import * as fontawesome from "@fortawesome/fontawesome";
import * as faCog from "@fortawesome/fontawesome-free-solid/faCog";
import * as faQuestion from "@fortawesome/fontawesome-free-solid/faQuestion";
import {HttpClientModule} from "@angular/common/http";
import {HistoryComponent} from './history/history.component';

import {
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatPaginatorModule,
    MatSortModule,
    MatTableModule
} from "@angular/material";
import {TradeDetailComponent} from './trade-detail/trade-detail.component';
import {TradeTableRowComponent} from './trade-table-row/trade-table-row.component';
import {TrailingProfitFormComponent} from './trailingprofitform/trailing-profit-form.component';

fontawesome.library.add(faCog);
fontawesome.library.add(faQuestion);

@NgModule({
    declarations: [
        AppComponent,
        TradeComponent,
        TrailingProfitFormComponent,
        StopLossFormComponent,
        WithQuoteAssetPipe,
        TradeTableComponent,
        TradeTableRowComponent,
        ConfigComponent,
        HistoryComponent,
        TradeDetailComponent,
    ],
    imports: [
        // Angular modules.
        BrowserModule,
        AppRoutingModule,
        FormsModule,
        ReactiveFormsModule,
        BrowserAnimationsModule,
        HttpClientModule,

        // Angular Material modules.
        MatButtonModule,
        MatPaginatorModule,
        MatTableModule,
        MatFormFieldModule,
        MatInputModule,
        MatSortModule,
    ],
    providers: [
        BinanceService,
        ToastrService,
        BinanceApiService,
    ],
    bootstrap: [
        AppComponent,
    ]
})
export class AppModule {
}
