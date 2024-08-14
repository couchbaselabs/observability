import { Component, Input, OnInit } from '@angular/core';
import { Cluster, heartBeatMessage } from 'src/app/types';

@Component({
  selector: 'app-cluster-status-summary',
  templateUrl: './cluster-status-summary.component.html',
  styleUrls: ['../../css/font-awesome.css'],
})
export class ClusterStatusSummaryComponent implements OnInit {
  @Input() cluster?: Cluster;
  @Input() count!: Map<string, number>;
  heartBeatMessage = heartBeatMessage;
  constructor() {}

  ngOnInit(): void {}

  runningCheckers() {
    return ['running', 'in progress'].includes(this.cluster?.status_progress?.status.toLocaleLowerCase() || '');
  }

  getPending() {
    if (!this.cluster?.status_progress) {
      return 0;
    }

    return (
      this.cluster.status_progress.total_checkers -
      this.cluster.status_progress.done -
      this.cluster.status_progress.failed
    );
  }
}
