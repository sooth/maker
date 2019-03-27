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

import {
    AfterViewInit,
    Component,
    Input,
    OnDestroy,
    OnInit
} from "@angular/core";
import {MakerService, TradeStatus} from "../maker.service";
import {Logger, LoggerService} from "../logger.service";
import {ToastrService} from "../toastr.service";
import {AppTradeState} from '../trade-table/trade-table.component';
import * as $ from "jquery";
import {BinanceApiService} from "../binance-api.service";

interface SellAtPriceModel {
    price: number;
}

interface SellAtPercentModel {
    percent: number;
}

const sellAtPercentModels: {
    [key: string]: SellAtPercentModel
} = {};

function getSellAtPercentModel(tradeId: string): SellAtPercentModel {
    if (sellAtPercentModels.hasOwnProperty(tradeId)) {
        return sellAtPercentModels[tradeId];
    }
    const model: SellAtPercentModel = {
        percent: 0,
    };
    sellAtPercentModels[tradeId] = model;
    return sellAtPercentModels[tradeId];
}

const sellAtPriceModels: {
    [key: string]: SellAtPriceModel
} = {};

function getSellAtPriceModel(tradeId: string, defaults: number[]): SellAtPriceModel {

    let price = 0;

    for (let defaultPrice of defaults) {
        if (defaultPrice > 0) {
            price = defaultPrice;
            break;
        }
    }

    if (sellAtPriceModels.hasOwnProperty(tradeId)) {
        const model = sellAtPriceModels[tradeId];
        if (!model.price) {
            model.price = price;
        }
        return model;
    }
    const model: SellAtPriceModel = {
        price: price,
    };
    sellAtPriceModels[tradeId] = model;
    return sellAtPriceModels[tradeId];
}

enum MARKET_SELL_STATE {
    INITIAL = "INITIAL",
    CONFIRM = "CONFIRM",
}

@Component({
    selector: "[app-trade-table-row]",
    templateUrl: "./trade-table-row.component.html",
    styleUrls: ["./trade-table-row.component.scss"]
})
export class TradeTableRowComponent implements OnInit, OnDestroy, AfterViewInit {

    MARKET_SELL_STATE = MARKET_SELL_STATE;

    TradeStatus = TradeStatus;

    logger: Logger = null;

    @Input("trade") trade: AppTradeState = null;

    @Input() showArchiveButtons: boolean = true;

    @Input() showTradeButtons: boolean = false;

    sellAtPriceModel: SellAtPriceModel = null;
    sellAtPercentModel: SellAtPercentModel = null;

    marketSellState = MARKET_SELL_STATE.INITIAL;

    abandonState = 0;

    destroyHooks: any[] = [];

    constructor(public maker: MakerService,
                private toastr: ToastrService,
                private binanceApi: BinanceApiService,
                logger: LoggerService) {
        this.logger = logger.getLogger("TradeTableRowComponent");
    }

    ngOnInit() {
        this.sellAtPercentModel = getSellAtPercentModel(this.trade.TradeID);
        this.sellAtPriceModel = getSellAtPriceModel(this.trade.TradeID, [
            this.trade.EffectiveBuyPrice,
            +(this.trade.BuyOrder.Price * (1 + 0.001)).toFixed(8),
        ]);
    }

    ngOnDestroy() {
        for (const hook of this.destroyHooks) {
            hook();
        }
    }

    ngAfterViewInit() {
        let sellDropdownHandler = $("#sellDropdown-" + this.trade.TradeID).on("hidden.bs.dropdown", () => {
            // Reset market sell confirmation state.
            this.marketSellState = MARKET_SELL_STATE.INITIAL;

            // Reset abandon state.
            this.abandonState = 0;
        });
        this.destroyHooks.push(() => {
            sellDropdownHandler.off();
        });
    }

    cancelBuy() {
        this.maker.cancelBuy(this.trade.TradeID).subscribe(() => {
        }, (error) => {
            console.log("Failed to cancel buy order: " + JSON.stringify(error));
        });
    }

    cancelSell() {
        this.maker.cancelSell(this.trade).subscribe(() => {
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

    onMarketSellClick($event: any) {
        if (this.marketSellState == MARKET_SELL_STATE.INITIAL) {
            this.marketSellState = MARKET_SELL_STATE.CONFIRM;
            $event.stopPropagation();
        } else {
            this.maker.marketSell(this.trade);
        }
    }

    archive() {
        this.maker.archiveTrade(this.trade);
    }

    abandon() {
        this.maker.abandonTrade(this.trade);
    }

}
