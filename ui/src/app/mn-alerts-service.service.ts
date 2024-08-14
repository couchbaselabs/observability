import { Injectable } from '@angular/core';

import { Alert } from './types';

@Injectable({
  providedIn: 'root',
})
export class MnAlertsService {
  alerts: Alert[] = [];
  constructor() {}

  success(message: string, timeout: number = 10000) {
    this.addAlert({
      message: message,
      type: 'success',
      timeout: timeout,
    });
  }

  error(message: string, timeout: number = 15000) {
    this.addAlert({
      message: message,
      type: 'error',
      timeout: timeout,
    });
  }

  warning(message: string, timeout: number = 15000) {
    this.addAlert({
      message: message,
      type: 'warning',
      timeout: timeout,
    });
  }

  private startTimer(item: Alert, timeout: number) {
    return setTimeout(() => {
      this.removeItem(item);
    }, timeout);
  }

  removeItem(item: Alert) {
    let index = this.alerts.indexOf(item);
    item.timeoutFn && clearTimeout(item.timeoutFn);
    this.alerts.splice(index, 1);
  }

  private addAlert(alert: Alert) {
    //in case we get alert with the same message
    //but different id find and remove it
    let foundItem = this.alerts.find((allAlerts) => {
      return alert.type == allAlerts.type && alert.message == allAlerts.message;
    });

    foundItem && this.removeItem(foundItem);
    alert.timeout && (alert.timeoutFn = this.startTimer(alert, alert.timeout));

    this.alerts.push(alert);
  }
}
