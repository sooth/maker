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

import {Component, EventEmitter, Input, OnInit, Output} from "@angular/core";
import {MakerService, TradeState, TradeStatus} from "../maker.service";
import {Logger, LoggerService} from "../logger.service";
import {ToastrService} from "../toastr.service";
import {AppTradeState} from '../trade-table/trade-table.component';

@Component({
    selector: "[app-trade-table-row]",
    templateUrl: "./trade-table-row.component.html",
    styleUrls: ["./trade-table-row.component.scss"]
})
export class TradeTableRowComponent implements OnInit {

    TradeStatus = TradeStatus;

    private logger: Logger = null;

    showJson = false;

    @Input("trade") trade: AppTradeState = null;

    @Output() symbolClickHandler: EventEmitter<any> = new EventEmitter();

    @Input() showArchiveButtons: boolean = true;

    @Input() showTradeButtons: boolean = false;

    sellAtPriceModel: {
        price: number;
    } = {
        price: null,
    };

    sellAtPercentModel: {
        percent: number;
    } = {
        percent: 0,
    };

    constructor(public maker: MakerService,
                private toastr: ToastrService,
                logger: LoggerService) {
        this.logger = logger.getLogger("TradeTableRowComponent");
    }

    ngOnInit() {
        this.sellAtPriceModel.price = this.trade.EffectiveBuyPrice;
    }

    cancelBuy(trade: TradeState) {
        this.maker.cancelBuy(trade);
    }

    cancelSell(trade: TradeState) {
        this.maker.cancelSell(trade).subscribe(() => {
        }, (error) => {
            this.toastr.error(`Failed to cancel sell order: ${error.error.message}`);
        });
    }

    limitSellAtPercent() {
        this.maker.limitSellByPercent(this.trade, +this.sellAtPercentModel.percent);
    }

    limitSellAtPrice() {
        this.maker.limitSellByPrice(this.trade, +this.sellAtPriceModel.price);
    }

    marketSell(trade: TradeState) {
        this.maker.marketSell(trade);
    }

    archive(trade: TradeState) {
        this.maker.archiveTrade(trade);
    }

    abandon(trade: TradeState) {
        this.maker.abandonTrade(trade);
    }

}
