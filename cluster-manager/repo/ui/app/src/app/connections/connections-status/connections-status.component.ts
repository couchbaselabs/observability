import { Component, OnInit, Input } from '@angular/core';

@Component({
  selector: 'app-connections-status',
  templateUrl: './connections-status.component.html',
  styleUrls: ['../../css/font-awesome.css'],
})

export class ConnectionsStatusComponent implements OnInit {
  @Input() conns!: number;
  constructor() {}

  ngOnInit(): void {}

  tooltipCheck(): string {
      if (this.conns >= 60000) {
        return `${this.conns} Connections to cluster. Limit is 60000 connections to cluster - Assess hosts and reduce
         connections.`;
      }

      if (this.conns >= 50000) {
        return `${this.conns} Connections to cluster. Approaching limit of 60000 - please assess hosts`;
      }

      return `${this.conns} Connections to cluster`;
  }
}
