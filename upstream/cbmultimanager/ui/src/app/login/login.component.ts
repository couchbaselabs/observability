import { Component, OnInit } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { Router } from '@angular/router';

import { AuthService } from '../auth.service';

@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  styleUrls: ['../css/modal.css', '../css/font-awesome.css', '../css/tooltip.css'],
})
export class LoginComponent implements OnInit {
  submitErr?: string;
  loginForm = this.fb.group({
    user: ['', Validators.required],
    password: ['', Validators.required],
  });

  constructor(private fb: FormBuilder, private authService: AuthService, private router: Router) {}

  ngOnInit(): void {}

  login() {
    this.submitErr = undefined;
    this.authService.logIn(this.loginForm.value).subscribe(
      () => {
        this.router.navigate([''], {});
      },
      (err) => {
        switch (err.status) {
          case 400:
            this.submitErr = 'Login failed. Please try again.';
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
