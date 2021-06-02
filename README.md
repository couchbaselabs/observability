# Observability

A work in progress, not intended for use by anyone or supported in any way.

The intention of this repository is to provide a simple, out-of-the-box solution based on industry standard tooling to observe the state of your Couchbase cluster.
* An additional requirement is to ensure we can integrate into existing observability pipelines people may already have as easily as possible.
* This must all support being deployed on-premise and on cloud platforms with minimal change.
* Any bespoke software must be minimal and ideally just restricted to configuration of generic tools.
* We must support customer configuration of what is "important" to monitor in their clusters although with best practice defaults provided.
* A simple and often upgrade pipeline to support frequent changes and updates to the solution which are then easy to roll out for users.

We essentially need to support two fairly distinct types of user:
1. Those who have nothing and just want a simple working solution to monitor their cluster.
2. Those who have an existing monitoring pipeline and want to integrate Couchbase monitoring into it, likely with a set of custom rules and configuration.

## Components

* Grafana: AGPL 3.0 https://github.com/grafana/grafana/blob/main/LICENSE
* Loki: AGPL 3.0 https://github.com/grafana/loki/blob/main/LICENSE
* Prometheus: Apache 2.0 https://github.com/prometheus/prometheus/blob/main/LICENSE
* Alert Manager: Apache 2.0 https://github.com/prometheus/alertmanager/blob/master/LICENSE
* Push Gateway: Apache 2.0 https://github.com/prometheus/pushgateway/blob/master/LICENSE
* Node Exporter: Apache 2.0 https://github.com/prometheus/node_exporter/blob/master/LICENSE
* Fluent Bit: Apache 2.0 https://github.com/fluent/fluent-bit/blob/master/LICENSE
* Couchbase Healthcheck: Proprietary/TBC

# Architecture

A Grafana-based stack has been selected for a few reasons:
* Already in use with Prometheus exporter and similar on Couchbase Server
* Industry standard and OSS with various integration points for other pipelines
* Existing options to scale from single node up to cloud scale easily

We expose Prometheus endpoints already for node-level information on a Couchbase Server instance.
Recently we have exposed logs using Fluent Bit, primarily for kubernetes based solutions but this can be deployed on-premise as well.

There is also a project underway to provide cluster-level information via a Prometheus endpoint: https://github.com/couchbaselabs/cbmultimanager
This will essentially integrate Couchbase knowledge and best practices into a reusable component. We can then reuse this component in the monitoring stack here as it provides a Prometheus endpoint.

Our observability stack is therefore a combination of Couchbase-specific components that provide that discrete knowledge and monitoring of the Couchbase cluster which are then plumbed into a generic Grafana pipline like below:

![Overview](/images/healthcheck-blocks.png)

A complete working stack is provided with a Grafana UI to access it all.

We also provide the various Prometheus endpoints for people who want to plumb it into another stack, or federate Prometheus instances or some other alternative.
They can reuse all the configuration provided, just not use Grafana for example.

Configuration points are also provided to tune the various alerts, dashboards and other configuration for a particular deployment, although basic deployment will provide a configured best practice version that is fully usable.

# Microlith deployment

To support easy deployment across a variety of targets, we are providing a 'microlith' single container option.
This is essentially the various scalable components of the Grafana stack (Loki, Prometheus, Grafana, Alert Manager) and Couchbase binaries for specific data extraction all runnable as a single multi-process container instance.

A single container can then be run on-premise or on a Kubernetes platform very easily with minimal effort.

Whilst on-premise customers may primarily be using native binaries, all supported OS's for Couchbase Server can run containers easily. This also makes it easier to deploy as a self-contained image and easy to upgrade as well. We could produce an OS-specific package (e.g. RPM) with all necessary dependencies on the container runtime.

# Distributed deployment

TBD - using the same approach as the PLG stack to pull apart the microlith and just provide the configuration to the distributed components.

For those customers who want to scale up the deployment and/or follow a more cloud-native approach using microservices that are easier to manage.

# Outstanding issues

* Configuration of the credentials required.
* Event log changes to provide node level information directly from files.
* Documentation and tutorials