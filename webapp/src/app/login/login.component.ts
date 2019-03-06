// Copyright (C) 2019 Cranky Kernel
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

import {Component, OnInit} from '@angular/core';
import {MakerApiService} from "../maker-api.service";
import {LoginService} from "../login.service";
import {Router} from "@angular/router";

@Component({
    selector: 'app-login',
    templateUrl: './login.component.html',
    styleUrls: ['./login.component.scss']
})
export class LoginComponent implements OnInit {

    username: string = "";
    password: string = "";

    error: boolean = false;

    constructor(private api: MakerApiService,
                private router: Router,
                private loginService: LoginService) {
    }

    ngOnInit() {
    }

    login() {
        this.loginService.login(this.username, this.password).subscribe((response) => {
            this.router.navigate(["/"]);
        }, (error) => {
            this.error = true;
        })
    }
}
