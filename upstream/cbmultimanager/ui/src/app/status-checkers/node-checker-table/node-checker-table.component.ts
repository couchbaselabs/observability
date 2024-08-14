import { Component, Input, OnInit, Output, EventEmitter } from '@angular/core';

import { Paginator } from 'src/app/paginator';
import { Checkers, NodeSummary, StatusResults } from 'src/app/types';
import { getStatusHealth } from '../../common';

@Component({
  selector: 'app-node-checker-table',
  templateUrl: './node-checker-table.component.html',
})
export class NodeCheckerTableComponent implements OnInit {
  expanded: Map<string, boolean> = new Map<string, boolean>();
  @Input() expandedResults!: Map<string, boolean>;
  @Output() expandedResultsChange = new EventEmitter<Map<string, boolean>>();
  @Output() dismissEvent = new EventEmitter<StatusResults>();
  @Input() filterByStatus!: (results: StatusResults[]) => StatusResults[];
  @Input() definitions: Checkers;
  @Input() statuses: StatusResults[];
  @Input() statusFilter: string[];
  @Input() set nodes(nodes: NodeSummary[]) {
    this.paginator.setContent(nodes);
  }

  get nodes(): NodeSummary[] {
    return this.paginator.getContent();
  }
  paginator: Paginator;
  getNodeStatusHealth = getStatusHealth;
  constructor() {
    this.statusFilter = [];
    this.statuses = [];
    this.definitions = {};
    this.paginator = new Paginator(10);
  }

  ngOnInit(): void {}

  getNodeStatusResults(nodeUUID: string): StatusResults[] {
    return (
      this.statuses.filter((result: StatusResults) => {
        return result.node === nodeUUID;
      }) || []
    );
  }

  nodeHealthTextColour(node: NodeSummary) {
    if (node.cluster_membership !== 'active' || node.status === 'unhealthy') {
      return 'error';
    }

    if (node.status === 'warm_up') {
      return 'warning';
    }

    return 'success';
  }

  dismiss(result: StatusResults) {
    this.dismissEvent.emit(result);
  }
}
