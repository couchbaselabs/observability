import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private token: string;
  public redirectUrl: string;

  private httpHeaders = new HttpHeaders({ 'Content-Type': 'application/json' });

  constructor(private http: HttpClient) {
    this.token = '';
    this.redirectUrl = '';
  }

  private setToken(token: any) {
    this.token = token;
    localStorage.setItem('cbmultimanager-token', this.token);
  }

  logOut(): void {
    this.token = '';
    localStorage.removeItem('cbmultimanager-token');
  }

  isLoggedIn(): boolean {
    return this.getToken().length > 0;
  }

  getToken(): string {
    if (this.token.length === 0) {
      this.token = localStorage.getItem('cbmultimanager-token') || '';
    }

    return this.token;
  }

  logIn(auth: { user: string; passoword: string }): Observable<any> {
    return this.http.post('/api/v1/self/token', auth, { headers: this.httpHeaders, responseType: 'text' }).pipe(
      map((token) => {
        this.setToken(token);
        return 200;
      })
    );
  }
}
