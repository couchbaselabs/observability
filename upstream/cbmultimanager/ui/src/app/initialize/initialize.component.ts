import { Component, OnInit } from '@angular/core';
import { FormBuilder, Validators, ValidatorFn, AbstractControl, ValidationErrors } from '@angular/forms';
import { Router } from '@angular/router';
import { Subscription } from 'rxjs';

import { InitService } from '../init.service';

@Component({
  selector: 'app-initialize',
  templateUrl: './initialize.component.html',
  styleUrls: ['../css/modal.css', '../css/font-awesome.css', '../css/tooltip.css'],
})
export class InitializeComponent implements OnInit {
  submitErr?: string;
  initForm = this.fb.group(
    {
      user: ['', Validators.required],
      password: ['', Validators.required],
      passwordConfirm: ['', Validators.required],
    },
    { validators: passwordMismatchValidator }
  );
  private initEvent!: Subscription;
  constructor(private fb: FormBuilder, private initService: InitService, private router: Router) {
    this.initEvent = this.initService.initialized.subscribe((init: boolean) => {
      if (init) {
        this.router.navigateByUrl(this.initService.redirectUrl);
      }
    });
  }

  ngOnInit(): void {}

  ngOnDestroy() {
    this.initEvent.unsubscribe();
  }

  submitForm() {
    let form = this.initForm.value;
    this.submitErr = undefined;

    this.initService.initialize({ user: form.user, password: form.password }).subscribe(
      () => {},
      (err) => {
        switch (err.status) {
          case 400:
            this.submitErr = 'Invalid user or password: ' + err.error.msg;
            break;
          case 500:
            this.submitErr = 'An internal server error caused login to fail. Please try again.';
            break;
          default:
            this.submitErr = 'Unexpected error. Please try again.';
        }
      }
    );
  }
}

const passwordMismatchValidator: ValidatorFn = (control: AbstractControl): ValidationErrors | null => {
  return control.get('password')?.value === control.get('passwordConfirm')?.value ? null : { passwordMismatch: true };
};
