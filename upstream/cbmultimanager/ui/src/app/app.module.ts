import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { HttpClientModule } from '@angular/common/http';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { NgbModule } from '@ng-bootstrap/ng-bootstrap';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { LoginComponent } from './login/login.component';
import { MainComponent } from './main/main.component';
import { ClustersComponent } from './clusters/clusters.component';
import { NodesTableComponent } from './clusters/nodes-table/nodes-table.component';
import { MnElementCraneModule } from './mn.element.crane';
import { AddClusterPanelComponent } from './clusters/add-cluster-panel/add-cluster-panel.component';
import { StatusCheckersComponent } from './status-checkers/status-checkers.component';
import { CheckerTableComponent } from './status-checkers/checker-table/checker-table.component';
import { BackButtonComponent } from './back-button.component';
import { ClusterStatusSummaryComponent } from './status-checkers/cluster-status-summary/cluster-status-summary.component';
import { StatusFilterComponent } from './status-checkers/status-filter/status-filter.component';
import { NodeCheckerTableComponent } from './status-checkers/node-checker-table/node-checker-table.component';
import { BucketsCheckerTableComponent } from './status-checkers/buckets-checker-table/buckets-checker-table.component';
import { FormatBytesPipe } from './format-bytes.pipe';
import { EditClusterPanelComponent } from './clusters/edit-cluster-panel/edit-cluster-panel.component';
import { InitializeComponent } from './initialize/initialize.component';
import { FormatVersionPipe } from './format-version.pipe';
import { DismissPanelComponent } from './status-checkers/dismiss-panel/dismiss-panel.component';
import { DismissalsComponent } from './dismissals/dismissals.component';
import { ConnectionsStatusComponent } from './connections/connections-status/connections-status.component';
import { ConnectionsComponent } from './connections/connections.component';
import { ClusterSubNavComponent } from './cluster-sub-nav/cluster-sub-nav.component';
import { LogsComponent } from './logs/logs.component';
import { OnpremComponent } from './clusters/onprem/onprem.component';
import { CloudComponent } from './clusters/cloud/cloud.component';
import { AddCredsComponent } from './clusters/cloud/add-creds/add-creds.component';
import { ClusterHealthComponent } from './clusters/cloud/cluster-health/cluster-health.component';
import { LinkifyPipe } from './linkify.pipe';

@NgModule({
  declarations: [
    AppComponent,
    LoginComponent,
    MainComponent,
    ClustersComponent,
    NodesTableComponent,
    AddClusterPanelComponent,
    StatusCheckersComponent,
    CheckerTableComponent,
    BackButtonComponent,
    ClusterStatusSummaryComponent,
    StatusFilterComponent,
    NodeCheckerTableComponent,
    BucketsCheckerTableComponent,
    FormatBytesPipe,
    EditClusterPanelComponent,
    InitializeComponent,
    FormatVersionPipe,
    DismissPanelComponent,
    DismissalsComponent,
    ConnectionsStatusComponent,
    ConnectionsComponent,
    ClusterSubNavComponent,
    LogsComponent,
    OnpremComponent,
    CloudComponent,
    AddCredsComponent,
    ClusterHealthComponent,
    LinkifyPipe,
  ],
  imports: [
    BrowserModule,
    MnElementCraneModule,
    HttpClientModule,
    FormsModule,
    ReactiveFormsModule,
    AppRoutingModule,
    NgbModule,
  ],
  providers: [],
  bootstrap: [AppComponent]
})
export class AppModule { }
