export class Paginator {
  public pageSize: number;
  private content: any[];
  public currentPage: number = 1;
  constructor(pageSize: number = 10) {
    this.pageSize = pageSize;
    this.content = [];
  }

  setContent(content: any[]) {
    this.content = content;
  }

  getContent(): any[] {
    return this.content;
  }

  contentLength(): number {
    return this.content.length;
  }

  getPage(): any[] {
    if (this.content.length === 0) {
      return [];
    }

    const start = (this.currentPage - 1) * this.pageSize;
    const end = start + this.pageSize < this.content.length ? start + this.pageSize : this.content.length;

    if (start >= this.content.length) {
      return [];
    }

    return this.content.slice(start, end);
  }
}
