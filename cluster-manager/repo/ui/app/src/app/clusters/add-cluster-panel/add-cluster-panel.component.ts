import { Component, OnInit } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { NgbActiveModal } from '@ng-bootstrap/ng-bootstrap';

import { RestService } from '../../rest.service';

@Component({
  selector: 'app-add-cluster-panel',
  templateUrl: './add-cluster-panel.component.html',
  styleUrls: ['../../css/modal.css'],
})
export class AddClusterPanelComponent implements OnInit {
  registerClusterForm = this.fb.group({
    host: ['', Validators.required],
    user: ['', [Validators.required, Validators.pattern(/^[A-Za-z0-9-_]{1,50}$/)]],
    password: ['', Validators.required],
  });
  submitting = false;
  submitErr?: string;

  constructor(public activeModal: NgbActiveModal, private fb: FormBuilder, private restService: RestService) { }

  ngOnInit(): void {
  }

  submitForm() {
    this.submitErr = undefined;
    this.submitting = true;
    this.restService.registerCluster(this.registerClusterForm.value).subscribe(
      (res) => {
        this.submitting = false;
        this.activeModal.close({worker: true});
      },
      (err) => {
        this.submitting = false;
        if ('error' in err) {
          this.submitErr = `${err.error.msg} - ${err.error.extras}`;
        } else {
          this.submitErr = 'Unexpected error, please try again later';
        }
      },
    );
  }
}
