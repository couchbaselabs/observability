import { Component, Input, Output, OnInit, EventEmitter } from '@angular/core';

import { Paginator } from 'src/app/paginator';
import { BucketSummary, Checkers, StatusResults } from 'src/app/types';
import { getStatusHealth } from '../../common';

@Component({
  selector: 'app-buckets-checker-table',
  templateUrl: './buckets-checker-table.component.html',
})
export class BucketsCheckerTableComponent implements OnInit {
  expanded: Map<string, boolean> = new Map<string, boolean>();
  @Input() expandedResults!: Map<string, boolean>;
  @Output() expandedResultsChange = new EventEmitter<Map<string, boolean>>();
  @Output() dismissEvent = new EventEmitter<StatusResults>();
  @Input() filterByStatus!: (results: StatusResults[]) => StatusResults[];
  @Input() definitions: Checkers;
  @Input() statuses: StatusResults[];
  @Input() statusFilter: string[];
  @Input() set buckets(buckets: BucketSummary[]) {
    this.paginator.setContent(buckets);
  }

  get buckets(): BucketSummary[] {
    return this.paginator.getContent();
  }
  paginator: Paginator;
  getBucketStatusHealth = getStatusHealth;
  constructor() {
    this.statusFilter = [];
    this.statuses = [];
    this.definitions = {};
    this.paginator = new Paginator(10);
  }

  ngOnInit(): void {}

  getBucketStatusResults(bucket: string): StatusResults[] {
    return (
      this.statuses.filter((result: StatusResults) => {
        return result.bucket === bucket;
      }) || []
    );
  }

  dismiss(result: StatusResults) {
    this.dismissEvent.emit(result);
  }
}
