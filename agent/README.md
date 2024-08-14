# cbhealthagent

cbhealthagent is an integrated health checker for Couchbase Server. Its design is minimal and lightweight. Currently, cbhealthagent runs like any other daemon/systemd process on a given Couchbase Server node. It listens over TCP and, when Prometheus asks cbhealthagent for statistics or metrics, cbhealthagent runs a 'collector' for a given service (for example memcached, GSI).

The collector service retrieves all the relevant metrics from a Couchbase Server node. If the metrics are not Prometheus compatible (as of Couchbase Server 7.0.x, metrics are in Prometheus format) the metrics are transformed with the [cmos-prometheus-exporter](https://github.com/couchbaselabs/cmos-prometheus-exporter/blob/main/pkg/metrics/defaultMetricSet.json). The metrics are then forwarded to Prometheus over TCP.

The high-level overview of cbhealthagent is:

* Fluent Bit: A C-binary that forwards all logs in the node's logging directory (default `/opt/couchbase/var/lib/couchbase/logs`) to the Hazelnut daemon over TCP. Cbhealthagent uses the [fluentbit Go wrapper](https://github.com/couchbaselabs/cbmultimanager/tree/master/agent/pkg/fluentbit) to configure Fluent Bit and accept the incoming log stream for further processing.
* [Hazelnut](https://github.com/couchbaselabs/cbmultimanager/tree/master/agent/pkg/hazelnut): Processes logs forwarded from Fluent Bit and acts on them according to a defined set of JSON rules (checkers). Review the [Hazelnut: Writing Rules](https://github.com/couchbaselabs/cbmultimanager/blob/master/agent/pkg/hazelnut/writing-rules.md) documentation for more details.
* [Exporter](https://github.com/couchbaselabs/cmos-prometheus-exporter): Transforms Couchbase Server 6.6.x metrics for compatibility with Prometheus.
* [Runner](https://github.com/couchbaselabs/cbmultimanager/tree/master/agent/pkg/health/runner): Gathers host-level statistics/configuration that do not come from Couchbase Server (for example from dmesg) and implements checkers based on the results.

By default, cbhealthagent waits for valid credentials to be passed from cluster-monitor although credentials can also be passed to cbhealthagent manually. This means that cbhealthagent can be run as a stand-alone process.

In the future, cbhealthagent is intended to be run by ns_server as a non-optional hidden service in the same way as goxdcr runs. For that purpose it is required that once build you put the resulting binary in the couchbase bin directory.

### Installing

This project requires go 1.16 or above. To build just do the following.

```
> go build -o ../build ./cmd/cbhealthagent
```

### Using

Once running it exposes 3 endpoints, this run on localhost so to access it externaly
it is required that it is proxied by ns_sever.

The endpoints are the following

```
GET /api/v1/checkers # Return all checker results
GET /api/v1/checkers/{name} # Returns only the checker 'name' result
GET /api/v1/ping # Checks that the agent is running
```