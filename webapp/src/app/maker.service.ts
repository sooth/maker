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
import {BehaviorSubject} from "rxjs/BehaviorSubject";
import {Subject} from "rxjs/Subject";
import {
    AccountInfo,
    AggTrade,
    BinanceApiService,
    buildAggTradeFromStream,
    CancelOrderResponse,
    RawStreamAccountInfo,
    StreamAggTrade
} from "./binance-api.service";
import {HttpParams} from "@angular/common/http";
import {Logger, LoggerService} from "./logger.service";
import {Observable} from "rxjs";
import {ToastrService} from './toastr.service';
import {LimitSellType} from './binance.service';
import {GIT_REVISION, VERSION} from "../environments/version";
import {take} from "rxjs/operators";
import {LoginService} from "./login.service";
import {MakerApiService} from "./maker-api.service";
import {MakerSocketService} from "./maker-socket.service";

export interface TradeMap {
    [key: string]: TradeState;
}

@Injectable({
    providedIn: "root"
})
export class MakerService {

    public tradeMap: TradeMap = {};

    public tradeMap$: BehaviorSubject<TradeMap> =
        new BehaviorSubject(this.tradeMap);

    public trade$: Subject<TradeState> = new Subject();

    public binanceAggTrades$: Subject<AggTrade> = new Subject();

    public binanceAccountInfo$: Subject<AccountInfo> = new Subject();

    private logger: Logger = null;

    constructor(logger: LoggerService,
                private toastr: ToastrService,
                private loginService: LoginService,
                private makerApi: MakerApiService,
                private binanceApi: BinanceApiService,
                private makerSocket: MakerSocketService) {
        this.logger = logger.getLogger("maker-service");
        this.makerSocket.$messages.subscribe((msg) => {
            this.onSocketMesasge(msg);
        });
        this.loginService.$onLogin.asObservable().pipe(take(1))
            .subscribe((result) => {
                this.init();
            });
    }

    private init() {
        this.makerSocket.start();
    }

    private onSocketMesasge(message: any) {
        switch (message.messageType) {
            case MakerMessageType.TRADE:
                this.onTrade(message.trade);
                break;
            case MakerMessageType.BINANCE_AGG_TRADE:
                const aggTrade = buildAggTradeFromStream(message.binanceAggTrade);
                this.binanceAggTrades$.next(aggTrade);
                break;
            case MakerMessageType.TRADE_ARCHIVED:
                delete (this.tradeMap[message.tradeId]);
                this.tradeMap$.next(this.tradeMap);
                break;
            case MakerMessageType.BINANCE_OUTBOUND_ACCOUNT_INFO:
                const accountInfo = AccountInfo.fromStream(
                    message.binanceOutboundAccountInfo);
                this.binanceAccountInfo$.next(accountInfo);
                break;
            case MakerMessageType.VERSION:
                this.checkVersion(<VersionMessage>message);
                break;
            case MakerMessageType.NOTICE:
                this.handleNotification(message.notice);
                break;
            default:
                this.logger.log(`Unhandled message type: ${message.messageType}`);
                this.logger.log(message);
                break;
        }
    }

    private handleNotification(notice: any) {
        console.log(notice);
        switch (notice.level) {
            case "error":
                this.toastr.error(notice.message, "Error", {
                    closeButton: true,
                    preventDuplicates: true,
                    timeOut: 10000,
                    progressBar: true,
                });
                break;
            case "info":
                this.toastr.info(notice.message, "", {
                    closeButton: true,
                    preventDuplicates: true,
                    timeOut: 10000,
                    progressBar: true,
                });
                break;
            default:
                this.toastr.warning(notice.message, "Warning", {
                    closeButton: true,
                    preventDuplicates: true,
                    timeOut: 10000,
                    progressBar: true,
                });
        }
    }

    private checkVersion(versionMesage: VersionMessage) {
        console.log(`Client version: ${VERSION}; Server version: ${versionMesage.version}.`);
        console.log(`Client git-rev: ${GIT_REVISION}; Server git-rev: ${versionMesage.git_revision}.`);
        if (VERSION != versionMesage.version || GIT_REVISION != versionMesage.git_revision) {
            this.toastr.warning("Backend has been updated. Reloading.", "", {
                progressBar: true,
                timeOut: 5000,
                closeButton: false,
                onHidden: () => {
                    location.reload();
                },
            });
        }
    }

    private onTrade(trade: TradeState) {
        this.tradeMap[trade.TradeID] = trade;
        this.tradeMap$.next(this.tradeMap);
        this.trade$.next(trade);
    }

    public updateStopLoss(trade: TradeState, enable: boolean, percent: number) {
        const params = new HttpParams()
            .set("enable", String(enable))
            .set("percent", percent.toFixed(8));
        this.makerApi.post(`/api/binance/trade/${trade.TradeID}/stopLoss`, null, {
            params: params,
        }).subscribe((response) => {
            console.log(response);
        });
    }

    public updateTrailingProfit(trade: TradeState, enable: boolean,
                                percent: number, deviation: number) {
        const params = new HttpParams()
            .set("enable", String(enable))
            .set("percent", percent.toFixed(8))
            .set("deviation", deviation.toFixed(8));
        this.makerApi.post(`/api/binance/trade/${trade.TradeID}/trailingProfit`, null, {
            params: params,
        }).subscribe((response) => {
        });
    }

    cancelBuy(trade: TradeState) {
        this.binanceApi.cancelBuy(trade.TradeID).subscribe((response) => {
        }, (error) => {
            console.log("Failed to cancel buy order: " + JSON.stringify(error));
        });
    }

    cancelSell(trade: TradeState): Observable<CancelOrderResponse> {
        return this.binanceApi.cancelSellOrder(trade.TradeID);
    }

    limitSellByPercent(trade: TradeState, percent: number) {
        this.logger.log(`Posting limit sell order at ${percent.toFixed(8)}%.`);
        const params = new HttpParams().set("percent", percent.toFixed(8));
        this.makerApi.post(`/api/binance/trade/${trade.TradeID}/limitSellByPercent`, null, {
            params: params,
        }).subscribe((response) => {
            if (response) {
                this.logger.log("Limit sell response: " + JSON.stringify(response));
            }
        }, (error) => {
            this.logger.log("Limit sell error: " + JSON.stringify(error));
            this.toastr.error(error.error, "Failed to post sell order.")
        });
    }

    limitSellByPrice(trade: TradeState, price: number) {
        const params = new HttpParams().set("price", price.toFixed(8));
        this.makerApi.post(`/api/binance/trade/${trade.TradeID}/limitSellByPrice`, null, {
            params: params,
        }).subscribe((response) => {
            if (response) {
                this.logger.log("Limit sell response: " + JSON.stringify(response));
            }
        }, (error) => {
            this.logger.log("Limit sell error: " + JSON.stringify(error));
            this.toastr.error(error.error, "Failed to post sell order.")
        });
    }

    marketSell(trade: TradeState) {
        this.makerApi.post(`/api/binance/trade/${trade.TradeID}/marketSell`, null, {}).subscribe((response) => {
            console.log(response);
        });
    }

    archiveTrade(trade: TradeState) {
        this.makerApi.post(`/api/binance/trade/${trade.TradeID}/archive`, null, {})
            .subscribe(() => {
            }, (error) => {
                this.logger.log("Failed to archive trade: " + JSON.stringify(error));
            });
    }

    abandonTrade(trade: TradeState) {
        this.makerApi.post(`/api/binance/trade/${trade.TradeID}/abandon`, null, {})
            .subscribe(() => {
            }, (error) => {
                this.logger.log("Failed to abandon trade: " + JSON.stringify(error));
            });
    }

    getVersion(): Observable<{
        opsys: string,
        arch: string,
        version: string,
        git_branch: string,
        git_revision: string,
    }> {
        return this.makerApi.get("/api/version");
    }

}

export enum TradeStatus {
    NEW = "NEW",
    FAILED = "FAILED",
    PENDING_BUY = "PENDING_BUY",
    WATCHING = "WATCHING",
    PENDING_SELL = "PENDING_SELL",
    DONE = "DONE",
    CANCELED = "CANCELED",
    ABANDONED = "ABANDONED",
}

export interface TradeState {
    TradeID: string;
    Symbol: string;
    Status: TradeStatus;
    OpenTime: string; // ISO format.
    CloseTime: string; // ISO format.
    Fee: number;
    BuyOrder: {
        Price: number;
        Quantity: number;
    };
    BuyFillQuantity: number;
    AverageBuyPrice: number;
    BuyCost: number;
    SellFillQuantity: number;
    AverageSellPrice: number;
    SellCost: number;
    StopLoss: {
        Enabled: boolean;
        Percent: number;
        Triggered: boolean;
    };
    TrailingProfit: {
        Enabled: boolean;
        Percent: number;
        Deviation: number;
        Activated: boolean;
        Triggered: boolean;
    };
    EffectiveBuyPrice: number;
    Profit: number;
    ProfitPercent: number;
    LastBuyStatus: string;
    LastSellstatus: string;
    LastPrice?: number;
    SellOrder: {
        Type: string;
        Status: string;
        Price: number;
        Quantity: number;
    };
    LimitSell: {
        Enabled: boolean;
        Type: LimitSellType;
        Percent: number;
        Price: number;
    }
    SellableQuantity: number;
}

export interface MakerMessage {
    messageType: string;
    trade?: TradeState;
    binanceAggTrade?: StreamAggTrade;
    tradeId?: string;
    binanceOutboundAccountInfo: RawStreamAccountInfo;
    notice?: any;
}

export enum MakerMessageType {
    VERSION = "version",
    NOTICE = "notice",
    TRADE = "trade",
    BINANCE_AGG_TRADE = "binanceAggTrade",
    TRADE_ARCHIVED = "tradeArchived",
    BINANCE_EXECUTION_REPORT = "binanceExecutionReport",
    BINANCE_OUTBOUND_ACCOUNT_INFO = "binanceOutboundAccountInfo",
}

interface VersionMessage extends MakerMessage {
    version: string,
    git_revision: string,
}
