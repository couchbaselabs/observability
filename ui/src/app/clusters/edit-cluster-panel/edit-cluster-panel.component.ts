import { Component, Input, OnInit } from '@angular/core';
import { AbstractControl, FormBuilder, ValidationErrors } from '@angular/forms';
import { NgbActiveModal } from '@ng-bootstrap/ng-bootstrap';
import { RestService } from 'src/app/rest.service';

@Component({
  selector: 'app-edit-cluster-panel',
  templateUrl: './edit-cluster-panel.component.html',
  styleUrls: ['../../css/modal.css'],
})
export class EditClusterPanelComponent implements OnInit {
  updateClusterForm = this.fb.group({
    host: [''],
    user: [''],
    password: [''],
  }, { validators: this.allFieldsRequired });
  submitting = false;
  submitErr?: string
  @Input() clusterUUID!: string;
  constructor(public activeModal: NgbActiveModal, private fb: FormBuilder, private restService: RestService) { }

  ngOnInit(): void {
  }

  submitForm() {
    this.submitting = true;
    const form = this.updateClusterForm.value;
    this.restService.editCluster(this.clusterUUID, form).subscribe(
      (res) => {
        this.submitting = false;
        this.activeModal.close({ worked: true });
      },
      (err) => {
        this.submitting = false;
        if ('error' in err) {
          this.submitErr = `${err.error.msg} - ${err.error.extras}`;
        } else {
          this.submitErr = 'Unexpected error, please try again later';
        }
      }
    );
  }

  allFieldsRequired(control: AbstractControl): ValidationErrors | null {
    const host: string = control.get('host')?.value || '';
    const user: string = control.get('user')?.value || '';
    const password: string = control.get('password')?.value || '';

    if (host.length === 0 && user.length === 0 && password.length === 0) {
      return { atLeastOneRequired: { valid: false } };
    }

    if ((user.length === 0 && password.length !== 0) || (user.length !== 0 && password.length === 0)) {
      return { userAndPasswordRequired: { valid: false } }
    }

    return null;
  }
}
