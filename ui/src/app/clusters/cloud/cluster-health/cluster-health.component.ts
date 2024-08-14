import { Component, Input, OnInit } from '@angular/core';
import { timer, Subscription } from 'rxjs';

import { getAPIError } from 'src/app/common';
import { RestService } from 'src/app/rest.service';
import { CloudCluster, CloudClusterHealth } from 'src/app/types';

@Component({
  selector: 'app-cluster-health',
  templateUrl: './cluster-health.component.html',
})
export class ClusterHealthComponent implements OnInit {
  @Input() cluster!: CloudCluster;
  clusterHealth?: CloudClusterHealth;
  clusterHealthErr?: string;
  cloudHealthSubscription!: Subscription;
  bucketSummary: boolean = true;
  nodeSummary: boolean = true;
  constructor(private rest: RestService) { }

  ngOnInit(): void {
    this.cloudHealthSubscription = timer(0, 30000).subscribe((_) => {
      this.rest.getCloudClusterHealth(this.cluster.id).subscribe(
        (res: CloudClusterHealth) => {
          this.clusterHealthErr = '';
          this.clusterHealth = res;
        },
        (err) => {
          this.clusterHealthErr = getAPIError(err);
        },
        );
    });
  }

  ngOnDestroy() {
    this.cloudHealthSubscription.unsubscribe();
  }

  getHealthColor(health: any): string {
    return (health === 'healthy')? 'dynamic_healthy': 'dynamic_inactive';
  }
}
