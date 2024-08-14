import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { NgbModal } from '@ng-bootstrap/ng-bootstrap';
import { interval, Subscription } from 'rxjs';

import { getAPIError } from '../common';
import { MnAlertsService } from '../mn-alerts-service.service';
import { Paginator } from '../paginator';
import { RestService } from '../rest.service';
import { Checkers, Cluster, ClusterStatusResults, StatusResults } from '../types';
import { DismissPanelComponent } from './dismiss-panel/dismiss-panel.component';

@Component({
  selector: 'app-status-checkers',
  templateUrl: './status-checkers.component.html',
  styleUrls: ['../css/font-awesome.css'],
})
export class StatusCheckersComponent implements OnInit {
  loading = false;
  clusterUUID: string;
  cluster?: Cluster;
  clusterResults?: ClusterStatusResults;
  count: Map<string, number>;
  definitions: Checkers;
  statusFilter: string[];
  bucketPaginator: Paginator;
  name: string = '';
  alias: string = '';
  resultSub!: Subscription;
  clusterSub!: Subscription;
  expandedResults = new Map<string, boolean>();

  constructor(
    private rest: RestService,
    private route: ActivatedRoute,
    private router: Router,
    private alertService: MnAlertsService,
    private modalService: NgbModal
  ) {
    this.clusterUUID = '';
    this.definitions = {};
    this.count = new Map<string, number>();
    this.statusFilter = this.getStatusFilters();
    this.bucketPaginator = new Paginator(10);
  }

  ngOnInit(): void {
    this.clusterUUID = this.route.snapshot.paramMap.get('clusterUUID') || '';
    if (this.clusterUUID?.length == 0) {
      return;
    }

    this.rest.getCheckers().subscribe(
      (res) => {
        this.definitions = res;
      },
      (err: HttpErrorResponse) => {
        this.alertService.error(getAPIError(err));
      }
    );

    this.getClusterStatus(this.clusterUUID, true);
    this.resultSub = interval(10000).subscribe(() => {
      this.getClusterStatus(this.clusterUUID);
    });

    this.getCluster(this.clusterUUID);
    this.clusterSub = interval(10000).subscribe(() => {
      this.getCluster(this.clusterUUID);
    })
  }

  ngOnDestroy(): void {
    this.resultSub.unsubscribe();
    this.clusterSub.unsubscribe();
  }

  getCluster(uuid: string) {
    this.rest.getCluster(uuid).subscribe(
      (cluster) => {
        this.cluster = cluster;
        this.alias = cluster.alias || '';
      },
      (err) => {
        this.alertService.error(getAPIError(err));
      },
    )
  }

  getClusterStatus(uuid: string, spinner: boolean = false) {
    this.loading = true && spinner;
    this.rest.getClusterStatus(uuid).subscribe(
      (cluster) => {
        this.name = cluster.name;
        this.loading = false;
        this.count = new Map<string, number>();

        this.bucketPaginator.setContent(cluster.buckets_summary);
        cluster.status_results.forEach((result: StatusResults) => {
          this.count.set(result.result.status, (this.count.get(result.result.status) || 0) + 1);
        });

        this.clusterResults = cluster;
      },
      (err: HttpErrorResponse) => {
        this.loading = false;
        this.alertService.error(getAPIError(err));
      }
    );
  }

  getClusterIdentifier(): string {
    return (this.alias)? this.alias : this.name || this.clusterUUID;
  }

  getClusterStatusResults(): StatusResults[] {
    return this.clusterResults?.status_results.filter((result: StatusResults) => (result.cluster.length > 0
      && !result.node && !result.log_file && !result.bucket)) || [];
  }

  filterByStatus(results: StatusResults[]): StatusResults[] {
    return (
      results.filter((res: StatusResults) => {
        return this.statusFilter.includes(res.result.status);
      }) || []
    );
  }

  getStatusFilters(): string[] {
    const queryStatusFilter = this.route.snapshot.queryParamMap.get('statusFilter');
    try {
      if (queryStatusFilter) {
        const statusFilter: string[] = JSON.parse(queryStatusFilter);
        return statusFilter;
      }
    } catch {
      console.error('Invalid status filter');
    }

    return ['warn', 'alert', 'info'];
  }

  updateStatusFilters(statusFilter: string[]) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { statusFilter: JSON.stringify(statusFilter) },
      queryParamsHandling: 'merge',
    });
    this.statusFilter = statusFilter;
  }

  refreshCluster(uuid: string) {
    this.rest.refreshCluster(uuid).subscribe(
      () => {
        this.alertService.success('Cluster refresh triggered. Rerunning checkers, this may take a couple of minutes');
      },
      (err) => {
        this.alertService.error(getAPIError(err));
      }
    );
  }

  dismiss(result: StatusResults) {
    let level = 'cluster';
    let containerID = '';
    if (result.bucket) {
      level = 'bucket';
      containerID = result.bucket;
    } else if (result.node) {
      level = 'node';
      containerID = result.node;
    }

    const modalRef = this.modalService.open(DismissPanelComponent);
    modalRef.componentInstance.level = level;
    modalRef.componentInstance.clusterUUID = result.cluster;
    modalRef.componentInstance.containerID = containerID;
    modalRef.componentInstance.checkerName = result.result.name;
    modalRef.result
      .then(() => {
        this.alertService.success('Checker dismissed');
        this.getClusterStatus(this.clusterUUID, true);
      })
      .catch((err) => {});
  }
}
