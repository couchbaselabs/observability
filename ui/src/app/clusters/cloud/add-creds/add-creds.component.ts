import { Component, OnInit } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { NgbActiveModal } from '@ng-bootstrap/ng-bootstrap';
import { RestService } from 'src/app/rest.service';

@Component({
  selector: 'app-add-creds',
  templateUrl: './add-creds.component.html',
  styleUrls: ['../../../css/modal.css']
})
export class AddCredsComponent implements OnInit {
  addCloudCredsForm = this.fb.group({
    name: ['', [Validators.required, Validators.maxLength(30)]],
    accessKey: ['', Validators.required],
    secret: ['', Validators.required],
  });
  submitting = false;
  submitErr?: string
  constructor(public activeModal: NgbActiveModal, private fb: FormBuilder, private restService: RestService) { }

  ngOnInit(): void {
  }

  get name() {
    return this.addCloudCredsForm.get('name');
  }

  get accessKey() {
    return this.addCloudCredsForm.get('accessKey');
  }

  get secret() {
    return this.addCloudCredsForm.get('secret');
  }

  submitForm() {
    this.submitting = true;
    const form = this.addCloudCredsForm.value;

    this.restService.addCloudCreds({name: form.name, access_key: form.accessKey, secret_key: form.secret}).subscribe(
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
      },
    )
  }

}
