import { Component, Input, OnInit, Output, EventEmitter } from '@angular/core';
import { Paginator } from 'src/app/paginator';
import { Checkers, StatusResults } from 'src/app/types';

@Component({
  selector: 'app-checker-table',
  templateUrl: './checker-table.component.html',
})
export class CheckerTableComponent implements OnInit {
  expanded: Map<string, boolean> = new Map<string, boolean>();
  @Input() set statuses(statuses: StatusResults[]) {
    this.paginator.setContent(statuses);
  }

  get statuses(): StatusResults[] {
    return this.paginator.getContent();
  }

  @Input() definitions!: Checkers;
  @Input() inner: boolean;
  @Input() expandedResults!: Map<string, boolean>;
  @Output() expandedResultsChange = new EventEmitter<Map<string, boolean>>();
  @Output() dismissEvent = new EventEmitter<StatusResults>();

  paginator: Paginator;
  constructor() {
    this.inner = false;
    this.paginator = new Paginator(10);
  }

  ngOnInit(): void {
    this.paginator.setContent(this.statuses);
  }

  statusToHealth(status: string): string {
    switch (status) {
      case 'good':
        return 'dynamic_healthy';
      case 'warn':
        return 'dynamic_warmup';
      case 'alert':
        return 'dynamic_unhealthy';
      default:
        return 'dynamic_inactive';
    }
  }

  getCheckerTitle(name: string): any {
    return this.definitions[name]?.title || name;
  }

  getCheckerDesription(name: string): string {
    return this.definitions[name]?.description || 'N/A';
  }

  isExpanded(result: StatusResults): boolean {
    return (
      this.expandedResults.get(
        `${result.cluster}-${result.bucket}-${result.node}-${result.log_file}-${result.result.name}`
      ) || false
    );
  }

  toggleExpand(result: StatusResults) {
    let key = `${result.cluster}-${result.bucket}-${result.node}-${result.log_file}-${result.result.name}`;
    this.expandedResults.set(key, !this.expandedResults.get(key));
    this.expandedResultsChange.emit(this.expandedResults);
  }

  dismiss(result: StatusResults) {
    this.dismissEvent.emit(result);
  }
}
