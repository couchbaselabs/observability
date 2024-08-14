import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'formatBytes'
})
export class FormatBytesPipe implements PipeTransform {
  transform(value: number, decimals: number = 2): string{
    const K = 1024;
    const M = K*K;
    const G = M*K;
    const T = G*K;
    const P = T*K;

    const units = [
      {val: P, postfix: 'PiB'},
      {val: T, postfix: 'TiB'},
      {val: G, postfix: 'GiB'},
      {val: M, postfix: 'MiB'},
      {val: K, postfix: 'KiB'},
    ];

    const t: {val: number, postfix: string} = units.find((unit: {val: number, postfix: string}) => {
      return value >= unit.val;
    }) || {val: 1, postfix: 'B'};

    return `${(value/ t.val).toFixed(decimals)}${t.postfix}`;
  }
}
