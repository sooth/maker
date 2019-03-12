// Copyright (C) 2018-2019 Cranky Kernel
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
import {MakerApiService} from "./maker-api.service";
import {ReplaySubject, Subject} from "rxjs";

export enum MakerSocketState {
    INITIALIZING = "initializing",
    CONNECTING = "connecting",
    CONNECTED = "connected",
    ERROR = "error",
    DISCONNECTED = "disconnected"
}

@Injectable({
    providedIn: 'root'
})
export class MakerSocketService {

    public state: MakerSocketState = MakerSocketState.INITIALIZING;

    private reconnects = 0;

    $messages = new Subject();

    stateChange$: ReplaySubject<string> = new ReplaySubject();

    constructor(private makerApi: MakerApiService) {
        this.stateChange$.next(this.state);
    }

    start() {
        this.connect();
    }

    private onMessage(msg: any) {
        this.$messages.next(msg);
    }

    private log(msg: string) {
        console.log("maker-socket: " + msg);
    }

    private setState(state: MakerSocketState) {
        this.state = state;
        this.stateChange$.next(this.state);
    }

    private connect() {
        this.state = MakerSocketState.CONNECTING;

        const ws = this.makerApi.openWebsocket();

        ws.onopen = () => {
            this.setState(MakerSocketState.CONNECTED);
            this.reconnects = 0;
            this.log("connected");
        };

        ws.onerror = () => {
            this.setState(MakerSocketState.ERROR);
            this.log("an error occurred, disconnecting");
            ws.close();
        };

        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.onMessage(message);
            } catch (err) {
                this.log("error: failed to parse message:");
                console.log(event);
            }
        };

        ws.onclose = () => {
            if (this.state !== MakerSocketState.ERROR) {
                this.setState(MakerSocketState.DISCONNECTED);
            }
            this.log("closed");
            this.reconnect();
        };
    }

    private reconnect() {
        if (this.reconnects > 1) {
            setTimeout(() => {
                this.connect();
            }, 1000);
        } else {
            this.connect();
        }
        this.reconnects++;
    }

}
