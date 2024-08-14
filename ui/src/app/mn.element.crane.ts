import {NgModule,
    Injectable,
    Component,
    ChangeDetectionStrategy,
    ElementRef,
    Renderer2,
    Input} from '@angular/core';

export {MnElementCraneModule,
    MnElementCraneService,
    MnElementCargoComponent,
    MnElementDepotComponent};

@Injectable()
class MnElementCraneService {
  depots: any;
  constructor() {
    this.depots = {};
  }

  register(element: any, name: string) {
    this.depots[name] = element;
  }

  unregister(name: string) {
    delete this.depots[name];
  }

  get(name: string): any {
    return this.depots[name];
  }
}

@Component({
  selector: "mn-element-cargo",
  template: "<ng-content></ng-content>",
  changeDetection: ChangeDetectionStrategy.OnPush,
})
class MnElementCargoComponent {
    @Input() depot!: any;
    depotElement: any;

  constructor(private el: ElementRef, private renderer: Renderer2,
      private mnElementCraneService :MnElementCraneService) {}

  ngOnInit() {
    this.depotElement = this.mnElementCraneService.get(this.depot);
    this.renderer.appendChild(this.depotElement.nativeElement, this.el.nativeElement);
  }

  ngOnDestroy() {
    this.renderer.removeChild(this.depotElement.nativeElement, this.el.nativeElement);
  }
}

@Component({
  selector: "mn-element-depot",
  template: "<ng-content></ng-content>",
  changeDetection: ChangeDetectionStrategy.OnPush
})
class MnElementDepotComponent {
  @Input() name!: string;
  constructor(private el: ElementRef, private mnElementCraneService: MnElementCraneService) {}

  ngOnInit() {
    this.mnElementCraneService.register(this.el, this.name);
  }

  ngOnDestroy() {
    this.mnElementCraneService.unregister(this.name);
  }
}

@NgModule({
  declarations: [
    MnElementDepotComponent,
    MnElementCargoComponent,
  ],
  exports: [
    MnElementDepotComponent,
    MnElementCargoComponent,
  ],
  entryComponents: [
    MnElementCargoComponent,
    MnElementDepotComponent,
  ],
  providers: [MnElementCraneService],
})
class MnElementCraneModule {}
