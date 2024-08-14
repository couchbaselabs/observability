import { HttpErrorResponse } from '@angular/common/http';
import { Component, Input, OnInit } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { NgbActiveModal } from '@ng-bootstrap/ng-bootstrap';

import { getAPIError } from '../../common';
import { RestService } from '../../rest.service';

@Component({
  selector: 'app-dismiss-panel',
  templateUrl: './dismiss-panel.component.html',
  styleUrls: ['../../css/modal.css'],
})
export class DismissPanelComponent implements OnInit {
  _level: string = 'cluster';
  showError: string = '';
  @Input() set level(lvl: string) {
    this._level = lvl;
    this.dismissCheckerForm.patchValue({ level: 0 });
  }
  @Input() clusterUUID: string = '';
  @Input() containerID: string = '';
  @Input() checkerName: string = '';
  submitting: boolean = false;
  levels: any = {
    cluster: ['All clusters', 'This cluster'],
    node: ['All nodes in all clusters', 'All nodes in this cluster', 'Only this node'],
    bucket: ['All buckets in all clusters', 'All buckets in this cluster', 'Only this bucket'],
  };
  dismissCheckerForm = this.fb.group({
    level: [0, Validators.required],
    forever: [false],
    scalar: [1, [Validators.min(1), Validators.max(400)]],
    unit: ['h'],
  });
  constructor(public activeModal: NgbActiveModal, private fb: FormBuilder, private restService: RestService) {}

  ngOnInit(): void {}

  dismiss(): void {
    this.showError = '';

    let form = this.dismissCheckerForm.value;
    let req: any = { checker_name: this.checkerName };

    if (form.forever) {
      req.forever = true;
    } else {
      let scalar = form.scalar;
      let unit = form.unit;
      if (unit === 'd') {
        scalar *= 24;
        unit = 'h';
      }

      req.dismiss_for = `${scalar}${unit}`;
    }

    // Dismiss levels  in the request are the following:
    // 0 - dismiss for everthing
    // 1 - dismiss only for the given cluster
    // 2 - dimiss only for the given bucket in the given cluster
    // 3 - dismiss only for the given node in the given cluster
    switch (form.level) {
      case 0:
        req.level = 0;
        break;
      case 1:
        req.level = 1;
        req.cluster_uuid = this.clusterUUID;
        break;
      case 2:
        req.cluster_uuid = this.clusterUUID;
        if (this._level === 'node') {
          req.level = 3;
          req.node_uuid = this.containerID;
        } else if (this._level === 'bucket') {
          req.level = 2;
          req.bucket_name = this.containerID;
        }
        break;
      default:
        this.showError = 'Invalid target selected';
        return;
    }

    this.restService.dismissChecker(req).subscribe(
      () => {
        this.activeModal.close();
      },
      (err: HttpErrorResponse) => {
        this.showError = getAPIError(err);
      }
    );
  }

  foreverChange() {
    this.dismissCheckerForm.patchValue({ forever: !this.dismissCheckerForm.get('forever')?.value });
  }

  isForever(): boolean | null {
    return this.dismissCheckerForm.get('forever')?.value ? true : null;
  }
}
