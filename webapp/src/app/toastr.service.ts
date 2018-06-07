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

import * as toastr from "toastr";

export interface ToastrOptions {
    progressBar?: boolean;
    timeOut?: number;
    extendedTimeOut?: number;
    closeButton?: boolean;
    preventDuplicates?: boolean;
    preventOpenDuplicates?: boolean;
}

@Injectable()
export class ToastrService {

    constructor() {
    }

    success(msg: string, title?: string, options?: ToastrOptions) {
        toastr.success(msg, title, options);
    }

    info(msg: string, title?: string, options?: ToastrOptions) {
        toastr.info(msg, title, options);
    }

    warning(msg: string, title?: string, options?: ToastrOptions) {
        toastr.warning(msg, title, options);
    }

    error(msg: string, title?: string, options?: ToastrOptions) {
        toastr.error(msg, title, options);
    }
}
