import { Component, OnInit } from '@angular/core';

import { getAPIError } from '../common';
import { MnAlertsService } from '../mn-alerts-service.service';
import { RestService } from '../rest.service';
import { Checkers, Dismissal } from '../types';

@Component({
  selector: 'app-dismissals',
  templateUrl: './dismissals.component.html',
  styleUrls: ['../css/font-awesome.css'],
})
export class DismissalsComponent implements OnInit {
  checkers: Checkers = {};
  dismissals: Map<string, Dismissal[]> = new Map<string, Dismissal[]>();
  disclosed: Map<string, boolean> = new Map<string, boolean>();
  loadingDismissals: boolean = false;
  constructor(private rest: RestService, private alertService: MnAlertsService) {}

  ngOnInit(): void {
    this.loadingDismissals = true;

    this.rest.getCheckers().subscribe(
      (res: Checkers) => {
        this.checkers = res;
        this.getDismissals();
      },
      (err) => {
        this.loadingDismissals = false;
        this.alertService.error(getAPIError(err));
      }
    );
  }

  getDismissals() {
    this.rest.getDismissals().subscribe(
      (res: Dismissal[]) => {
        let temp = new Map<string, Dismissal[]>();
        res.forEach((value: Dismissal) => {
          const identifier = this.getCheckerTitle(value.checker_name);
          if (temp.has(identifier)) {
            temp.get(identifier)?.push(value);
          } else {
            temp.set(identifier, [value]);
          }
        });

        this.dismissals = temp;
        this.loadingDismissals = false;
      },
      (err) => {
        this.loadingDismissals = false;
        this.alertService.error(getAPIError(err));
      }
    );
  }

  getCheckerTitle(name: string): string {
    if (!this.checkers || !this.checkers[name]) {
      return name;
    }

    return this.checkers[name].title;
  }

  getUntil(dismissal: Dismissal): string {
    return dismissal.forever ? 'forever' : dismissal.until || 'N/A';
  }

  getDismissalTarget(dismissal: Dismissal): string {
    switch (dismissal.level) {
      case 0:
        // Dismissed for everthing
        return 'All clusters';
      case 1:
        // Dissmised just for this cluster
        return `Cluster ${dismissal.cluster_uuid}`;
      case 2:
        // Dismissed just for one bucket in one cluster
        return `Bucket ${dismissal.bucket_name} of cluster ${dismissal.cluster_uuid}`;
      case 3:
        // Dismissed just for one node in one cluster
        return `Node ${dismissal.node_uuid} of cluster ${dismissal.cluster_uuid}`;
      default:
        return 'N/A';
    }
  }

  deleteDismissal(id: string) {
    this.rest.deleteDismissal(id).subscribe(
      () => {
        this.alertService.success('Dismissal removed');
        this.loadingDismissals = true;
        this.getDismissals();
      },
      (err) => {
        this.alertService.error(getAPIError(err));
      }
    );
  }
}
