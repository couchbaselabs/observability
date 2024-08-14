# cluster-monitor

The cluster-monitor (also referred to as cbmultimanager) is responsible for orchestrating the monitoring of one or more Couchbase Clusters. The cluster-monitor process regularly pings both the cbhealthagent process and the cluster (for example with REST APIs, `cbstats`, etc.) and pulls in metrics from these sources.

Some more complex checkers are implemented based on processing these metrics.

Some key components of cluster-monitor are:

1. [SingleClusterManager](https://github.com/couchbaselabs/cbmultimanager/blob/master/cluster-monitor/pkg/manager/cluster_manager.go): Responsible for all the processes that deal with a particular cluster. For example, heartbeats, checker runs, communication with cbhealthagent.
2. [Manager]((https://github.com/couchbaselabs/cbmultimanager/blob/master/cluster-monitor/pkg/manager/manager.go): Responsible for spawning the SingleClusterManagers, as well as serving the API.
3. [CheckExecutor](https://github.com/couchbaselabs/cbmultimanager/blob/master/cluster-monitor/pkg/status/check_executor.go): All checker runs are routed through a single CheckExecutor. This ensures that we're not overloading any clusters or repeating work.
