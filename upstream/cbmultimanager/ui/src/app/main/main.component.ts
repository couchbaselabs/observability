import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';

import { MnAlertsService } from '../mn-alerts-service.service';

@Component({
  selector: 'app-main',
  templateUrl: './main.component.html',
  styleUrls: ['../css/font-awesome.css'],
})
export class MainComponent implements OnInit {
  showRespMenu: boolean;
  showNav: boolean;
  removeAlert = this.alertService.removeItem;
  get alerts() {
    return this.alertService.alerts;
  }

  constructor(private router: Router, private alertService: MnAlertsService) {
    this.showRespMenu = true;
    this.showNav = true;

    if (router.url === '/') {
      router.navigate(['clusters'], { queryParamsHandling: 'preserve' });
    }
  }

  ngOnInit(): void {}
}
