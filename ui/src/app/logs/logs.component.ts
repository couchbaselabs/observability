import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { FormBuilder, Validators } from '@angular/forms';

import { Cluster } from '../types';
import { MnAlertsService } from '../mn-alerts-service.service';
import { RestService } from '../rest.service';
import { getAPIError } from '../common';

@Component({
  selector: 'app-logs',
  templateUrl: './logs.component.html',
})
export class LogsComponent implements OnInit {
  logs = [
    'audit',
    'couchdb',
    'error',
    'http_access',
    'json_rpc',
    'ns_couchdb',
    'ssl_proxy',
    'views',
    'babysitter',
    'debug',
    'goxdcr',
    'info',
    'metakv',
    'reports',
    'stats',
    'xdcr_target',
  ];
  submitting = false;
  clusterUUID = '';
  cluster?: Cluster;
  loading = false;
  logForm = this.fb.group({
    logFile: ['error', Validators.required],
    node: [null, Validators.required],
  });
  logOut: string = '';
  constructor(
    private restService: RestService,
    private alertService: MnAlertsService,
    private route: ActivatedRoute,
    private fb: FormBuilder
  ) {}

  ngOnInit(): void {
    this.clusterUUID = this.route.snapshot.paramMap.get('clusterUUID') || '';
    if (this.clusterUUID.length === 0) {
      return;
    }

    this.loading = true;
    this.restService.getCluster(this.clusterUUID).subscribe(
      (res: Cluster) => {
        this.loading = false;
        this.cluster = res;
        this.logForm.patchValue({ node: this.cluster.nodes_summary[0].node_uuid });
      },
      (err) => {
        this.loading = false;
        this.alertService.error(`Could not get cluster nodes: ${getAPIError(err)}`);
      }
    );
  }

  getClusterIdentifier(): string {
    if (!this.cluster) {
      return this.clusterUUID;
    }

    return this.cluster.name || this.cluster.uuid;
  }

  getLogFile() {
    this.submitting = true;
    const form = this.logForm.value;
    this.restService.getLogFile(this.clusterUUID, form.node, form.logFile).subscribe(
      (res) => {
        this.logOut = res;
        this.submitting = false;
      },
      (err) => {
        this.alertService.error(`Could not get log file: ${getAPIError(err)}`);
        this.submitting = false;
      }
    );
  }
}
