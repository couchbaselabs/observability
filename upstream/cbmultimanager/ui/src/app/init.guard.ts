import { Injectable } from '@angular/core';
import {
  CanActivate,
  ActivatedRouteSnapshot,
  RouterStateSnapshot,
  UrlTree,
  CanActivateChild,
  Router,
} from '@angular/router';
import { Observable } from 'rxjs';

import { InitService } from './init.service';

@Injectable({
  providedIn: 'root',
})
export class InitGuard implements CanActivate, CanActivateChild {
  constructor(private initService: InitService, private router: Router) {}

  canActivate(
    route: ActivatedRouteSnapshot,
    state: RouterStateSnapshot
  ): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
    return this.checkInit(state.url);
  }

  canActivateChild(
    route: ActivatedRouteSnapshot,
    state: RouterStateSnapshot
  ): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
    return this.canActivate(route, state);
  }

  checkInit(url: string): true | UrlTree {
    if (this.initService.isInitialized()) {
      return true;
    }

    this.initService.redirectUrl = url;
    return this.router.parseUrl('/init');
  }
}
