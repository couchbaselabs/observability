import { Component, Input, OnInit, Output, EventEmitter } from '@angular/core';

@Component({
  selector: 'app-status-filter',
  templateUrl: './status-filter.component.html',
})
export class StatusFilterComponent implements OnInit {
  @Input() statusFilter!: string[];
  @Output() statusChange = new EventEmitter<string[]>();
  constructor() {}

  ngOnInit(): void {}

  statusChanged(filterTerm: string) {
    const index = this.statusFilter.indexOf(filterTerm);
    if (index >= 0) {
      this.statusFilter.splice(index, 1);
    } else {
      this.statusFilter.push(filterTerm);
    }

    this.statusChange.emit(this.statusFilter);
  }
}
