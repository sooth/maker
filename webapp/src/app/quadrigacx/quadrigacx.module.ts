import {NgModule} from '@angular/core';
import {CommonModule} from '@angular/common';
import {QuadrigacxComponent} from './quadrigacx.component';
import {QuadrigacxService} from './quadrigacx.service';
import {HttpClientModule} from '@angular/common/http';
import {FormsModule} from '@angular/forms';

@NgModule({
    imports: [
        CommonModule,
        HttpClientModule,
        FormsModule,
    ],
    declarations: [
        QuadrigacxComponent,
    ],
    providers: [
        QuadrigacxService,
    ]
})
export class QuadrigacxModule {
}
