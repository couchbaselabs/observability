import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { AuthGuard } from './auth.guard';
import { LoginComponent } from './login/login.component';
import { MainComponent } from './main/main.component';
import { ClustersComponent } from './clusters/clusters.component'
import { StatusCheckersComponent } from './status-checkers/status-checkers.component';
import { InitializeComponent } from './initialize/initialize.component';
import { InitGuard } from './init.guard';
import { DismissalsComponent } from './dismissals/dismissals.component';
import { ConnectionsComponent } from './connections/connections.component';
import { LogsComponent } from './logs/logs.component';

const routes: Routes = [
  { path: 'init', component: InitializeComponent, pathMatch: 'full'},
  { path: 'login', component: LoginComponent, pathMatch: 'full' , canActivate: [InitGuard]},
  { path: '', component: MainComponent, canActivate: [InitGuard, AuthGuard], children: [
    { path: 'clusters', component: ClustersComponent, canActivateChild: [InitGuard, AuthGuard]},
    { path: 'dismissals', component: DismissalsComponent, canActivateChild: [InitGuard, AuthGuard]},
    { path: 'clusters/:clusterUUID/status', component: StatusCheckersComponent, canActivateChild: [InitGuard, AuthGuard]},
    { path: 'clusters/:clusterUUID/connections', component: ConnectionsComponent, canActivateChild: [InitGuard, AuthGuard]},
    { path: 'clusters/:clusterUUID/logs', component: LogsComponent, canActivateChild: [InitGuard, AuthGuard]},
  ]},
];

@NgModule({
  imports: [
    RouterModule.forRoot(routes),
  ],
  exports: [RouterModule]
})
export class AppRoutingModule { }
