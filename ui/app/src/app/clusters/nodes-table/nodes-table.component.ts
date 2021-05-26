import { Component, Input, OnInit } from '@angular/core';
import { Paginator } from 'src/app/paginator';

import { NodeSummary } from 'src/app/types';

@Component({
  selector: 'app-nodes-table',
  templateUrl: './nodes-table.component.html',
})
export class NodesTableComponent implements OnInit {

  @Input() nodes!: NodeSummary[];
  paginator: Paginator;
  constructor() {
    this.paginator = new Paginator(5);
  }

  ngOnInit(): void {
    this.paginator.setContent(this.nodes);
  }

  getNodeHealth(node: NodeSummary): string {
    if (node.cluster_membership !== 'active') {
      return 'dynamic_unhealthy';
    }

    return `dynamic_${node.status}`;
  }
}
