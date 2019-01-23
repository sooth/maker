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
import {GIT_BRANCH, VERSION} from "../../environments/version";
import {HttpClient, HttpParams} from "@angular/common/http";
import {MakerService} from "../maker.service";
import * as $ from "jquery";

@Component({
    selector: 'app-about',
    templateUrl: './about.component.html',
    styleUrls: ['./about.component.scss']
})
export class AboutComponent implements OnInit {

    VERSION = VERSION;

    updateAvailable: boolean | null = null;

    releaseBranch = GIT_BRANCH;

    constructor(private http: HttpClient, private maker: MakerService) {
    }

    ngOnInit() {
        $("#aboutModal").on("show.bs.modal", () => {
            this.updateAvailable = null;
        });
    }

    checkVersion() {
        this.maker.getVersion().subscribe((myVersion) => {
            let params = new HttpParams()
                .set("version", this.VERSION)
                .set("opsys", myVersion.opsys)
                .set("arch", myVersion.arch);
            this.http.get("https://maker.crankykernel.com/files/versions.json", {
                params: params,
            }).subscribe((latestVersion) => {
                this.compareVersions(myVersion, latestVersion);
            });
        });
    }

    compareVersions(myVersion, latestVersion) {
        if (myVersion.git_branch == "master") {
            const channel = "development";
            try {
                const commit_id = latestVersion[channel][myVersion.opsys][myVersion.arch].commit_id;
                if (myVersion.git_revision != commit_id) {
                    console.log("Development release has been updated.");
                    this.updateAvailable = true;
                } else {
                    this.updateAvailable = false;
                }
            } catch (err) {
                console.log("Failed to check if development release channel has update: " + err);
                console.log(`- myVersion: ${JSON.stringify(myVersion)}`);
                console.log(`- latestVersion: ${JSON.stringify(latestVersion)}`);
            }
            return;
        }

        // Check release channel.
        try {
            const version = latestVersion["release"][myVersion.opsys][myVersion.arch].version;
            if (version != myVersion.version) {
                console.log("Release version has been updated.");
                this.updateAvailable = true;
            } else {
                this.updateAvailable = false;
            }
        } catch (err) {
            console.log("Failed to check if release channel has update: " + err);
            console.log(`- myVersion: ${JSON.stringify(myVersion)}`);
            console.log(`- latestVersion: ${JSON.stringify(latestVersion)}`);
        }

    }

}
