import { HttpErrorResponse } from '@angular/common/http';
import { StatusResults } from './types';

export function getStatusHealth(statuses: StatusResults[], bucket: string = '', node: string = ''): string {
  if (!statuses) {
    return 'dynamic_inactive';
  }

  let warn = false;
  let found = false;

  for (let result of statuses) {
    if (bucket !== '' && result.bucket !== bucket) {
      continue;
    }

    if (node !== '' && result.node != node) {
      continue;
    }

    found = true;
    switch (result.result.status) {
      case 'alert':
        return 'dynamic_unhealthy';
      case 'warn':
        warn = true;
        break;
    }
  }

  if (!found) {
    return 'dynamic_inactive';
  }

  return warn ? 'dynamic_warmup' : 'dynamic_healthy';
}

export function getAPIError(err: HttpErrorResponse) {
  if (!err.error) {
    return err.message;
  }

  let msg: string = '';
  if (typeof err.error === 'string') {
    let error = JSON.parse(err.error);
    msg = 'extras' in error ? `${error.msg} - ${error.extras}` : error.msg;
  } else {
    msg = 'extras' in err.error ? `${err.error.msg} - ${err.error.extras}` : err.error.msg;
  }

  return msg;
}
