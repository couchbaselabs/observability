import { LinkifyPipe } from './linkify.pipe';
import { TestBed } from "@angular/core/testing";
import { DomSanitizer } from '@angular/platform-browser';
import { SecurityContext } from '@angular/core';

describe('LinkifyPipe', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [LinkifyPipe] });
  });

  it('creates A tags for URLs', () => {
    const pipe = TestBed.inject(LinkifyPipe);
    const domSanitizer = TestBed.inject(DomSanitizer);
    expect(
      domSanitizer.sanitize(SecurityContext.HTML, pipe.transform("https://example.com"))
    ).toEqual(`<a href="https://example.com" target="_blank" rel="noopener noreferrer">example.com</a>`)
  });

  it('creates A tags for JIRA references', () => {
    const pipe = TestBed.inject(LinkifyPipe);
    const domSanitizer = TestBed.inject(DomSanitizer);
    expect(
      domSanitizer.sanitize(SecurityContext.HTML, pipe.transform("See MB-12345", true))
    ).toEqual(`See <a href="https://issues.couchbase.com/browse/MB-12345" target="_blank" rel="noopener noreferrer">MB-12345</a>`)
  });
});
