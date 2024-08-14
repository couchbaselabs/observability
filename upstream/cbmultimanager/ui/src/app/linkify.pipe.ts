import { Pipe, PipeTransform } from '@angular/core';
import { DomSanitizer } from '@angular/platform-browser';
import AutoLinker from "autolinker";

function linkifyJiras(input: string) {
  return input.replace(/(^|[\s\.])([A-Z]+-\d+)([\s\.]|$)/g, `$1<a href="https://issues.couchbase.com/browse/$2" target="_blank" rel="noopener noreferrer">$2</a>$3`)
}

@Pipe({
  name: 'linkify'
})
export class LinkifyPipe implements PipeTransform {

  public constructor(private readonly domSanitizer: DomSanitizer) {}

  private linkifier = new AutoLinker({
    email: false,
    hashtag: false,
    mention: false,
    phone: false,
    sanitizeHtml: true
  });

  transform(value: string, jira = false) {
    let html = this.linkifier.link(value);
    if (jira) {
      html = linkifyJiras(html);
    }
    return this.domSanitizer.bypassSecurityTrustHtml(html);
  }

}
