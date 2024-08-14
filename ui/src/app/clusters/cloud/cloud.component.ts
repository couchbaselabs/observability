import { Component, OnInit } from '@angular/core';
import { NgbModal } from '@ng-bootstrap/ng-bootstrap';
import { getAPIError } from 'src/app/common';
import { MnAlertsService } from 'src/app/mn-alerts-service.service';
import { RestService } from 'src/app/rest.service';
import { CloudCluster, CloudClusterPage } from 'src/app/types';
import { AddCredsComponent } from './add-creds/add-creds.component';

@Component({
  selector: 'app-cloud',
  templateUrl: './cloud.component.html',
  styleUrls: ['../../css/font-awesome.css'],
})
export class CloudComponent implements OnInit {
  pageSize: number = 10;
  page: number = 1;
  cursor?: any;
  clusters: CloudCluster[] = [];
  loading: boolean = false;
  hasCloud: boolean = false;

  constructor(private rest: RestService, private alertService : MnAlertsService, private modalService: NgbModal) {}

  ngOnInit(): void {
    this.getCloudClusters();
  }

  getCloudClusters() {
    this.loading = true
    this.rest.getCloudClusters({ page: this.page, size: this.pageSize, sortBy: 'name' }).subscribe(
      (data: CloudClusterPage) => {
        this.cursor = data.cursor;
        this.clusters = data.data;
        this.hasCloud = true;
        this.loading = false;
      },
      (err) => {
        this.loading = false;
        // If we get a 400 it means we don't have cluster credentials yet. If that is the case then do nothing until
        // we get them.
        if (err.status === 400) {
          return
        }

        this.alertService.error(getAPIError(err));
      },
    );
  }

  pageChange() {
    this.getCloudClusters();
  }

  openAddCredsModal() {
    this.modalService.open(AddCredsComponent).result.then(() => {
      this.getCloudClusters();
    })
  }
}

