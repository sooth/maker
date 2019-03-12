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
import {Subject} from "rxjs";

enum State {
    INITIALIZING = "initializing",
    CONNECTING = "connecting",
    CONNECTED = "connected",
    ERRORED = "errored",
    DISCONNECTED = "disconnected"
}

@Injectable({
    providedIn: 'root'
})
export class MakerSocketService {

    public state: State = State.INITIALIZING;

    private reconnects = 0;

    $messages = new Subject();

    constructor(private makerApi: MakerApiService) {
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

    private connect() {
        this.state = State.CONNECTING;

        const ws = this.makerApi.openWebsocket();

        ws.onopen = () => {
            this.state = State.CONNECTED;
            this.reconnects = 0;
            this.log("connected");
        };

        ws.onerror = () => {
            this.state = State.ERRORED;
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
            if (this.state !== State.ERRORED) {
                this.state = State.DISCONNECTED;
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
