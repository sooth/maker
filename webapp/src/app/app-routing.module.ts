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

import {NgModule} from "@angular/core";
import {RouterModule, Routes} from "@angular/router";
import {TradeComponent} from "./trade/trade.component";
import {ConfigComponent} from "./config/config.component";
import {HistoryComponent} from './history/history.component';
import {TradeDetailComponent} from './trade-detail/trade-detail.component';
import {LoginComponent} from "./login/login.component";
import {CompositeGuard} from "./composite.guard";

const routes: Routes = [
    {
        path: "trade",
        component: TradeComponent,
        canActivate: [
            CompositeGuard,
        ],
        data: {
            authRequired: true,
            configRequired: true,
        }
    },
    {
        path: "trade/:tradeId",
        component: TradeDetailComponent,
        canActivate: [
            CompositeGuard,
        ],
        data: {
            authRequired: true,
            configRequired: true,
        }
    },
    {
        path: "config",
        component: ConfigComponent,
        canActivate: [
            CompositeGuard,
        ],
        data: {
            authRequired: true,
            configRequired: false,
        }
    },
    {
        path: "history",
        component: HistoryComponent,
        canActivate: [
            CompositeGuard,
        ],
        data: {
            authRequired: true,
            configRequired: true,
        }
    },
    {
        path: "login",
        pathMatch: "full",
        component: LoginComponent,
    },
    {
        path: "",
        pathMatch: "full",
        redirectTo: "trade",
    },
];

@NgModule({
    imports: [
        RouterModule.forRoot(routes),
    ],
    exports: [
        RouterModule
    ]
})
export class AppRoutingModule {
}
