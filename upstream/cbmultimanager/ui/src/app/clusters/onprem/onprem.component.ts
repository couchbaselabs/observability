import { Component, OnInit } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { NgbModal } from '@ng-bootstrap/ng-bootstrap';
import { interval, Subject, Subscription } from 'rxjs';
import { debounceTime } from 'rxjs/operators';

import { Cluster, NodeSummary, heartBeatMessage } from '../../types';
import { Paginator } from '../../paginator';
import { RestService } from '../../rest.service';
import { MnAlertsService } from '../../mn-alerts-service.service';
import { getAPIError } from '../../common';
import { EditClusterPanelComponent } from '../edit-cluster-panel/edit-cluster-panel.component';
import { FormatVersionPipe } from '../../format-version.pipe';

@Component({
  selector: 'app-onprem',
  templateUrl: './onprem.component.html',
  styleUrls: ['../../css/font-awesome.css'],
})
export class OnpremComponent implements OnInit {
  clusters?: Cluster[];
  loading?: boolean;
  expanedCluster: Map<string, boolean>;
  hasClusters = false;
  paginator: Paginator;
  heartBeatMessage = heartBeatMessage;
  sub!: Subscription;
  sortField: string = 'Critical';
  sortAscending: boolean = false;
  searchTerms: string = '';
  searchTermSubject = new Subject<string>();
  searchSub!: Subscription;

  constructor(private rest: RestService, private modalService: NgbModal, private alertService: MnAlertsService) {
    this.expanedCluster = new Map<string, boolean>();
    this.paginator = new Paginator(10);
    this.alertService = alertService;
  }

  ngOnInit(): void {
    this.getClusters();
    this.sub = interval(30000).subscribe(() => {
      this.getClusters(false);
    });

    // the search will match by name, version and or service. The syntax [name:val version:version  service:a] can be
    // used in that case the value after the : will only be matched with the given field.
    this.searchSub = this.searchTermSubject.pipe(debounceTime(500)).subscribe((searchTerm: string) => {
      this.searchTerms = searchTerm;
      this.paginator.setContent(this.sortClusters(this.filterValues(this.clusters)));
    });
  }

  filterValues(clusters?: Cluster[]): Cluster[] {
    if (!clusters) {
      return [];
    }

    if (this.searchTerms === '') {
      return clusters;
    }

    // build search query
    const parts = this.searchTerms.split(' ').map((part) => part.trim());

    // perform filtering
    return clusters.filter((cluster) => {
      for (const part of parts) {
        // name partial match
        if (cluster.name.includes(part)) {
          return true;
        }

        for (const node of cluster.nodes_summary) {
          // service name full match
          if (node.services.includes(part)) {
            return true;
          }

          // host partial match
          if (node.host.includes(part)) {
            return true;
          }

          // version partial match
          if (node.version.includes(part)) {
            return true;
          }
        }
      }

      return false;
    });
  }

  ngOnDestroy(): void {
    this.sub.unsubscribe();
    this.searchSub.unsubscribe();
  }

  getClusters(spinner: boolean = true) {
    this.loading = true && spinner;
    this.rest.getClusters().subscribe(
      (clusters: Cluster[]) => {
        this.clusters = clusters;
        this.paginator.setContent(this.sortClusters(this.filterValues(this.clusters)));
        this.hasClusters = clusters.length > 0;
        this.loading = false;
      },
      (err: HttpErrorResponse) => {
        this.hasClusters = false;
        this.alertService.error(getAPIError(err));
        this.loading = false;
      }
    );
  }

  deleteCluster(uuid: string) {
    this.rest.deleteCluster(uuid).subscribe(
      () => {
        this.getClusters();
        this.alertService.success('Cluster deleted');
      },
      (err) => {
        this.alertService.error(getAPIError(err));
      }
    );
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


  openEditClusterModal(clusterUUID: string) {
    const modalRef = this.modalService.open(EditClusterPanelComponent);
    modalRef.componentInstance.clusterUUID = clusterUUID;
    modalRef.result.then(() => {
      this.alertService.success('Cluster updated');
    });
  }

  getClusterIdentifier(cluster: Cluster): string {
    if (cluster.alias) {
      return `${cluster.name || '-'} (${cluster.alias})`;
    }

    return `${cluster.name || '-'}`;
  }

  getHost(cluster: Cluster): string {
    if (cluster.nodes_summary.length === 0) {
      return 'N/A';
    }

    return cluster.nodes_summary[0].host;
  }

  getStatusCounter(cluster: Cluster, status: string): (number | string) {
    if (!cluster.enterprise) {
      return 'N/A';
    }

    if (!cluster.status_summary) {
      return 0;
    }

    switch (status) {
      case 'alerts':
        return cluster.status_summary.alerts;
      case 'warnings':
        return cluster.status_summary.warnings;
      case 'good':
        return cluster.status_summary.good;
      default:
        return 0;
    }
  }

  getHealth(cluster: Cluster): string {
    if (!cluster.status_summary) {
      return 'dynamic_inactive';
    }

    if (cluster.status_summary.alerts > 0) {
      return 'dynamic_unhealthy';
    }

    if (cluster.status_summary.warnings > 0) {
      return 'dynamic_warmup';
    }

    return 'dynamic_healthy';
  }

  getVersions(cluster: Cluster): string {
    const version: Set<string> = new Set<string>(
      cluster.nodes_summary.map((node: NodeSummary) => {
        return FormatVersionPipe.formatVersion(node.version);
      })
    );

    return Array.from(version).join(', ') || 'Unknown';
  }

  getServices(cluster: Cluster): string[] {
    const services = new Set<string>();
    cluster.nodes_summary.forEach((node: NodeSummary) => {
      node.services.forEach((service: string) => services.add(service));
    });

    return Array.from(services);
  }

  updateSortBy(value: any) {
    this.sortField = value;

    if (this.clusters) {
      this.paginator.setContent(this.sortClusters(this.filterValues(this.clusters)));
    }
  }

  sortClusters(clusters?: Cluster[]): Cluster[] {
    if (!clusters) {
      return [];
    }

    switch (this.sortField) {
      case 'Name':
        clusters.sort((a, b) => this.sortByValue(a.name, b.name));
        break;
      case 'Critical':
        clusters.sort((a, b) =>
          this.sortByValue(this.getStatusCounter(a, 'alerts'), this.getStatusCounter(b, 'alerts'))
        );
        break;
      case 'Warnings':
        clusters.sort((a, b) =>
          this.sortByValue(this.getStatusCounter(a, 'warnings'), this.getStatusCounter(b, 'warnings'))
        );
        break;
      case 'Nodes':
        clusters.sort((a, b) => this.sortByValue(a.nodes_summary.length, b.nodes_summary.length));
        break;
      case 'Versions':
        clusters.sort((a, b) => {
          const aVersions = a.nodes_summary
            .map((node: NodeSummary) => node.version)
            .sort()
            .reverse();
          const bVersions = b.nodes_summary
            .map((node: NodeSummary) => node.version)
            .sort()
            .reverse();

          return this.sortByValue(aVersions[0], bVersions[0]);
        });
        break;
    }

    if (!this.sortAscending) {
      clusters.reverse();
    }

    return clusters;
  }

  sortByValue(a: any, b: any): number {
    if (a === b) {
      return 0;
    }

    return a < b ? -1 : 1;
  }

  sortOrderToggle() {
    this.sortAscending = !this.sortAscending;
    if (this.clusters) {
      this.paginator.setContent(this.sortClusters(this.filterValues(this.clusters)));
    }
  }
}
