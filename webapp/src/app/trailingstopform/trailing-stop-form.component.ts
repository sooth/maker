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

import {Component, ElementRef, Input, OnInit} from '@angular/core';
import {FormBuilder, FormGroup, Validators} from '@angular/forms';

import * as $ from "jquery";
import {MakerService, TradeState, TradeStatus} from '../maker.service';

@Component({
    selector: 'app-trailingstopform',
    templateUrl: './trailing-stop-form.component.html',
    styleUrls: ['./trailing-stop-form.component.scss']
})
export class TrailingStopFormComponent implements OnInit {

    TradeStatus = TradeStatus;

    @Input() trade: TradeState = null;

    form: FormGroup;

    constructor(private fb: FormBuilder,
                private el: ElementRef,
                private maker: MakerService) {
    }

    ngOnInit() {
        this.buildForm();
        if (this.trade.Status === TradeStatus.DONE) {
            $(this.el.nativeElement).find("input").attr("disabled", "disabled");
        }
    }

    private buildForm() {
        if (!this.trade.TrailingStop) {
            this.trade.TrailingStop.Enabled = false;
            this.trade.TrailingStop.Percent = 0;
            this.trade.TrailingStop.Deviation = 0;
        }
        this.form = this.fb.group({
            enabled: [this.trade.TrailingStop.Enabled,],
            percent: [this.trade.TrailingStop.Percent, Validators.required,],
            deviation: [this.trade.TrailingStop.Deviation, Validators.required,],
        })
    }

    onSubmit() {
        const formModel: FormModel = this.form.value;
        this.maker.updateTrailingStop(this.trade, formModel.enabled,
                +formModel.percent, +formModel.deviation);
    }

    reset() {
        this.buildForm();
    }
}

interface FormModel {
    enabled: boolean;
    percent: number;
    deviation: number,
}