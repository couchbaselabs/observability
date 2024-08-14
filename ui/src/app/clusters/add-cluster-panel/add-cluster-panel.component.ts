import { Component, OnInit } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { NgbActiveModal } from '@ng-bootstrap/ng-bootstrap';

import { RestService } from '../../rest.service';

@Component({
  selector: 'app-add-cluster-panel',
  templateUrl: './add-cluster-panel.component.html',
  styleUrls: ['../../css/modal.css', '../../css/font-awesome.css'],
})
export class AddClusterPanelComponent implements OnInit {
  registerClusterForm = this.fb.group({
    host: ['', Validators.required],
    user: ['', [Validators.required, Validators.pattern(/^((?![\(\)<>,;:\\"/\[\]\?=\{\}\s]).)*$/)]],
    alias: ['', [Validators.pattern(/^a-.+$/), Validators.maxLength(100)]],
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

    const formValue = this.registerClusterForm.value;
    if (formValue.alias !== '' && !formValue.alias.startsWith('a-')) {
      this.submitErr = `Aliases must start with 'a-'`;
      return;
    }

    this.restService.registerCluster(formValue).subscribe(
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
