import { Component, Input } from '@angular/core';
import { Router } from '@angular/router';

@Component({
  template: `
    <mn-element-cargo depot="actions">
      <div class="header-controls margin-right-half">
        <a (click)="goBack()"><span class="icon fa-arrow-left"></span> BACK</a>
      </div>
    </mn-element-cargo>
  `,
  selector: 'mn-back-btn',
  styleUrls: ['./css/font-awesome.css']
})
class BackButtonComponent {
  @Input() navigate!: string;
  constructor(private router: Router) {}

  goBack() {
    this.router.navigate([this.navigate], {
      queryParamsHandling: 'preserve',
    })
  }
}

export { BackButtonComponent };
