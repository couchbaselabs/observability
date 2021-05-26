export type NodeSummary = {
  node_uuid: string;
  version: string;
  host: string;
  status: string;
  cluster_membership: string;
  services: string[];

  expanded?: boolean;
};

export type ClusterStatusSummary = {
  good: number;
  warnings: number;
  alerts: number;
  info: number;
  missing: number;
};

export type StatusProgress = {
  status: string;
  done: number;
  failed: number;
  total_checkers: number;
  start?: string;
  end?: string;
};

export type Cluster = {
  uuid: string;
  name: string;
  nodes_summary: NodeSummary[];
  status_summary?: ClusterStatusSummary;
  heart_beat_issue?: number;
  last_update: string;
  status_progress?: StatusProgress;
};

export type ClusterAddRequest = {
  host: string;
  user: string;
  password: string;
};

export type BucketSummary = {
  name: string;
  storage_backend: string;
  quota: number;
  quota_used: number;
  num_replicas: number;
  items: number;
  bucket_type: string;

  expanded?: boolean;
};

export type ClusterStatusResults = {
  nodes_summary: NodeSummary[];
  buckets_summary: BucketSummary[];
  last_update: string;
  name: string;
  uuid: string;
  status_results: StatusResults[];
};

export type StatusResults = {
  cluster: string;
  node?: string;
  bucket?: string;
  log_file?: string;
  result: {
    name: string;
    remediation?: string;
    status: string;
    time: string;
    version: number;
    value?: any;
  };

  expanded?: boolean;
};

export type CheckerDefinition = {
  name: string;
  description: string;
  title: string;
  type: number;
};

export interface Checkers {
  [key: string]: CheckerDefinition;
}

export type Alert = {
  message: string;
  type: string;
  timeout: number;
  timeoutFn?: any;
};

export type Dismissal = {
  level: Number;
  id: string;
  checker_name: string;
  cluster_uuid?: string;
  bucket_name?: string;
  node_uuid?: string;
  forever?: boolean;
  until?: string;
};

export type Connections = {
  internal: number;
  connections: Connection[];
}

export type Connection = {
  agent_name: string;
  connection: string;
  socket: string;
  peer_name: string;
  ssl: boolean;
  sdk_name: string;
  sdk_version: string;
  total_send: number;
  total_recv: number;
  user: any;
  source: Address;
  target: Address;
};

export type Address = {
  ip: string;
  port: number;
}

export function heartBeatMessage(issue: number): string {
  switch (issue) {
    case 0:
      return 'none';
    case 1:
      return (
        'The user and password given are no longer valid or do not have the required permissions.' +
        'Please update them.'
      );
    case 2:
      return (
        'Could not establish connection with the cluster during last heartbeat. Please check the cluster is ' +
        'still online.'
      );
    case 3:
      return 'The given host no longer points to the same cluster. The cluster UUID has changed.';
    default:
      return `Unknown heart beat issue - ${issue}.`;
  }
}
