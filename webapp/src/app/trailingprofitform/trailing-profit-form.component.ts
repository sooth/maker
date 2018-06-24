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
    selector: 'app-trailingprofitform',
    templateUrl: './trailing-profit-form.component.html',
    styleUrls: ['./trailing-profit-form.component.scss']
})
export class TrailingProfitFormComponent implements OnInit {

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
        if (!this.trade.TrailingProfit) {
            this.trade.TrailingProfit.Enabled = false;
            this.trade.TrailingProfit.Percent = 0;
            this.trade.TrailingProfit.Deviation = 0;
        }
        this.form = this.fb.group({
            enabled: [this.trade.TrailingProfit.Enabled,],
            percent: [this.trade.TrailingProfit.Percent, Validators.required,],
            deviation: [this.trade.TrailingProfit.Deviation, Validators.required,],
        })
    }

    onSubmit() {
        const formModel: FormModel = this.form.value;
        this.maker.updateTrailingProfit(this.trade, formModel.enabled,
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