import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'formatVersion',
})
export class FormatVersionPipe implements PipeTransform {
  transform(value: string | null | undefined): string {
    return FormatVersionPipe.formatVersion(value);
  }

  static formatVersion(value: string | null | undefined): string {
    if (!value) {
      return 'Unknown';
    }

    // Expected format is A.B.C-XYZW-<enterprise|community>
    let parts = value.split('-');
    return parts.length == 3 ? `${parts[0]} ${parts[2].charAt(0).toUpperCase()}${parts[2].slice(1)}` : value;
  }
}
