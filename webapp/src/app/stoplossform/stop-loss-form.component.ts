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

import {Component, Input, OnInit} from '@angular/core';
import {FormBuilder, FormGroup} from '@angular/forms';
import {MakerService, TradeState, TradeStatus} from '../maker.service';

@Component({
    selector: 'app-stoploss-form',
    templateUrl: './stop-loss-form.component.html',
    styleUrls: ['./stop-loss-form.component.scss']
})
export class StopLossFormComponent implements OnInit {

    TradeStatus = TradeStatus;

    @Input() trade: TradeState = null;

    form: FormGroup;

    constructor(private fb: FormBuilder,
                private maker: MakerService) {
    }

    ngOnInit() {
        this.buildForm();
    }

    private buildForm() {
        this.form = this.fb.group({
            enabled: [this.trade.StopLoss.Enabled,],
            percent: [this.trade.StopLoss.Percent,],
        });
    }

    onSubmit() {
        const formModel: FormModel = this.form.value;
        this.maker.updateStopLoss(this.trade,
                formModel.enabled, +formModel.percent);
    }

    reset() {
        this.buildForm();
    }
}

interface FormModel {
    enabled: boolean;
    percent: number;
}