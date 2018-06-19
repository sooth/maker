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

import {AfterViewInit, Component, OnInit, ViewChild} from '@angular/core';
import {HttpClient} from '@angular/common/http';
import {TradeState} from '../maker.service';
import {
    MatPaginator,
    MatSort,
    MatSortable,
    MatTableDataSource
} from '@angular/material';
import {toAppTradeState} from '../trade-table/trade-table.component';

@Component({
    selector: 'app-history',
    templateUrl: './history.component.html',
    styleUrls: ['./history.component.scss']
})
export class HistoryComponent implements OnInit, AfterViewInit {

    displayedColumns = ["closeTime", 'symbol', "status", "profitPercent"];

    dataSource: MatTableDataSource<TradeState>;
    @ViewChild(MatPaginator) paginator: MatPaginator;
    @ViewChild(MatSort) sort: MatSort;

    constructor(private http: HttpClient) {
    }

    ngOnInit() {
    }

    ngAfterViewInit() {
        this.http.get("/api/trade/query")
                .subscribe((trades: TradeState[]) => {
                    this.dataSource = new MatTableDataSource(trades.map((trade) => {
                        return toAppTradeState(trade)
                    }));
                    this.dataSource.paginator = this.paginator;
                    this.dataSource.sort = this.sort;
                    this.sort.sort(<MatSortable>{
                        id: "CloseTime",
                        start: "desc",
                        disableClear: false,
                    });
                });
    }

    applyFilter(filterValue: string) {
        filterValue = filterValue.trim().toLowerCase();
        this.dataSource.filter = filterValue;
        if (this.dataSource.paginator) {
            this.dataSource.paginator.firstPage();
        }
    }

}
