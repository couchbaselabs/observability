import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';

import { getAPIError } from '../common';
import { MnAlertsService } from '../mn-alerts-service.service';
import { Paginator } from '../paginator';
import { RestService } from '../rest.service';
import { Cluster, Connection, Connections } from '../types';

@Component({
  selector: 'app-connections',
  templateUrl: './connections.component.html',
})
export class ConnectionsComponent implements OnInit {
  connectionsByAgent: any[] = [];
  expanded = new Map();
  clusterUUID: string = '';
  clusterName: string = '';
  loading: boolean = false;
  paginator: Paginator;
  totalConns: number = 0;
  constructor(private restService: RestService, private route: ActivatedRoute, private alertService: MnAlertsService) {
    this.paginator = new Paginator(10);
  }

  ngOnInit(): void {
    this.clusterUUID = this.route.snapshot.paramMap.get('clusterUUID') || '';
    if (this.clusterUUID?.length == 0) {
      return;
    }

    this.restService.getCluster(this.clusterUUID).subscribe((res: Cluster) => {
      this.clusterName = res.name;
    });

    this.loading = true;
    this.getConnections();
  }

  getConnections() {
    this.restService.getClusterConnections(this.clusterUUID).subscribe(
      (connections: Connections) => {
        this.loading = false;
        this.totalConns = connections.connections.length;

        // group by agent
        const temp = new Map<string, any>();
        connections.connections.forEach((conn: Connection) => {
          if (!temp.has(conn.agent_name)) {
            temp.set(conn.agent_name, {
              sdk: conn.sdk_name,
              version: conn.sdk_version,
              sources: new Set([conn.source?.ip]),
              targets: new Set([conn.target?.ip]),
              conns : 1,
            });
          } else {
            let val = temp.get(conn.agent_name);
            val.sources.add(conn.source?.ip);
            val.targets.add(conn.target?.ip);
            val.conns ++;
            temp.set(conn.agent_name, val)
          }
        });

        this.connectionsByAgent = Array.from(temp).sort();
        this.paginator.setContent(this.connectionsByAgent);
      },
      (err) => {
        this.loading = false;
        this.alertService.error(getAPIError(err));
      }
    );
  }

  getIPS(conn: any): string {
    return Array.from(conn.sources).join(', ')
  }

  getClusterIdentifier(): string {
    return (this.clusterName) ? this.clusterName : this.clusterUUID;
  }
}
