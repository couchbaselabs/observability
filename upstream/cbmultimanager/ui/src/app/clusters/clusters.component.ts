import { Component, OnInit, ViewChild } from '@angular/core';
import { NgbModal } from '@ng-bootstrap/ng-bootstrap';

import { MnAlertsService } from '../mn-alerts-service.service';
import { AddClusterPanelComponent } from './add-cluster-panel/add-cluster-panel.component';
import { OnpremComponent } from './onprem/onprem.component';

@Component({
  selector: 'app-clusters',
  templateUrl: './clusters.component.html',
})
export class ClustersComponent implements OnInit {
  @ViewChild(OnpremComponent)
  private child!: OnpremComponent;
  onPrem: boolean = true;
  cloud: boolean = false;
  constructor(private modalService: NgbModal, private alertService: MnAlertsService) {}

  ngOnInit(): void {}

  openAddClusterModal() {
    this.modalService.open(AddClusterPanelComponent).result.then(() => {
      this.child.getClusters();
      this.alertService.success('Cluster added');
    });
  }
}
