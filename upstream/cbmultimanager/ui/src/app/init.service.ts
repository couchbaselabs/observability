import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, BehaviorSubject } from 'rxjs';
import { tap } from 'rxjs/operators';

@Injectable({
  providedIn: 'root',
})
export class InitService {
  public redirectUrl: string;
  public initialized = new BehaviorSubject<boolean>(false);
  private httpHeaders = new HttpHeaders({ 'Content-Type': 'application/json' });
  private init: boolean = false;

  constructor(private http: HttpClient, private router: Router) {
    this.redirectUrl = 'clusters';
    this.http.get('/api/v1/self').subscribe((init: any) => {
      this.initialized.next(init.init || false);
    });

    this.initialized.subscribe((init: boolean) => {
      this.init = init;
    });
  }

  initialize(auth: any): Observable<any> {
    return this.http.post('/api/v1/self', auth, { headers: this.httpHeaders }).pipe(
      tap((res) => {
        this.initialized.next(true);
      })
    );
  }

  isInitialized(): boolean {
    return this.init;
  }

  handle503() {
    this.redirectUrl = this.router.url;
    this.initialized.next(false);
    this.router.navigateByUrl('/init');
  }
}
