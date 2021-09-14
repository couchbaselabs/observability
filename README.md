# Observability

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

This software uses the following components with their associated licensing also captured:

* Grafana: AGPL 3.0 https://github.com/grafana/grafana/blob/main/LICENSE
* Loki: AGPL 3.0 https://github.com/grafana/loki/blob/main/LICENSE
* Prometheus: Apache 2.0 https://github.com/prometheus/prometheus/blob/main/LICENSE
* Alert Manager: Apache 2.0 https://github.com/prometheus/alertmanager/blob/master/LICENSE
* Fluent Bit: Apache 2.0 https://github.com/fluent/fluent-bit/blob/master/LICENSE
* Jaeger: Apache 2.0 https://github.com/jaegertracing/jaeger/blob/master/LICENSE
* Nginx: https://github.com/nginxinc/docker-nginx/blob/master/LICENSE
* Prometheus Merge Tool: Apache 2.0 https://github.com/lablabs/prometheus-alert-overrider/blob/master/LICENSE
* Couchbase Cluster Monitor: Proprietary to Couchbase https://github.com/couchbaselabs/cbmultimanager/blob/master/LICENSE

Nginx is used as the base image for the microlith container.

All licences are in the source repository and the microlith container in the [`/licenses`](microlith/licenses/) directory.

A statement is printed out to standard output/console at start up to indicate acceptance of licensing and where you can find them all.

A simple [helper script](./tools/build-oss-container.sh) is provided as well to build without the Couchbase Cluster monitor.

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

![Microlith overview](/images/microlith-runtime.png)

Whilst on-premise customers may primarily be using native binaries, all supported OS's for Couchbase Server can run containers easily. This also makes it easier to deploy as a self-contained image and easy to upgrade as well. We could produce an OS-specific package (e.g. RPM) with all necessary dependencies on the container runtime.

For full details refer to the [microlith](microlith/README.md) sub-directory.

## On-premise usage

A working example is [provided](examples/native/) based on a docker compose stack to run up a single node Couchbase cluster with the microlith all correctly configured.

The basic steps are:
1. Install a container runtime for your platform, for example on Ubuntu details are here: https://docs.docker.com/engine/install/ubuntu/
2. Run the microlith container up: `docker run --name=couchbase-grafana --rm -d -P couchbase-observability`
3. Configure the cluster to talk to it by providing credentials to Prometheus and cluster monitor tools.

Prometheus end points and credentials can be added to the [config file](microlith/dynamic/prometheus/couchbase/targets.json) mounted into the container above. This is periodically rescanned and new end points added.

The cluster monitor currently requires configuration via a bespoke REST API:
`curl -u "${CLUSTER_MONITOR_USER}:${CLUSTER_MONITOR_PWD}" -X POST -d '{ "user": "'"${COUCHBASE_USER}"'", "password": "'"${COUCHBASE_PWD}"'", "host": "'"${COUCHBASE_ENDPOINT}"'" }' "${CLUSTER_MONITOR_ENDPOINT}/api/v1/clusters"`
* COUCHBASE_ENDPOINT should be set to a node that you want to monitor in a Couchbase cluster.
* COUCHBASE_USER & COUCHBASE_PWD are the credentials for accesing that cluster.
* CLUSTER_MONITOR_ENDPOINT is the mapping to port 7196 of the container we started, e.g. `http://localhost:7196`. In the container run line above we map to dynamic ports so grab them using `docker container port couchbase-grafana 7196` and use that value in the URL.
* CLUSTER_MONITOR_USER & CLUSTER_MONITOR_PWD are the credentials for the cluster monitor tool, defaults to a `admin:password` but can be set differently using these variables when launching the container.

As an example to configure a new cluster node to be monitored:
```
CLUSTER_MONITOR_USER=admin
CLUSTER_MONITOR_PWD=password
CLUSTER_MONITOR_ENDPOINT=http://localhost:$(docker container port couchbase-grafana 7196)
COUCHBASE_USER=Administrator
COUCHBASE_PWD=password
COUCHBASE_ENDPOINT=http://db2:8091
curl -u "${CLUSTER_MONITOR_USER}:${CLUSTER_MONITOR_PWD}" -X POST -d '{ "user": "'"${COUCHBASE_USER}"'", "password": "'"${COUCHBASE_PWD}"'", "host": "'"${COUCHBASE_ENDPOINT}"'" }' "${CLUSTER_MONITOR_ENDPOINT}/api/v1/clusters"
```
We can also run with a directory containing shell scripts that do the above: `-v $PWD/microlith/dynamic/healthcheck/:/etc/healthcheck/`
This will be re-scanned periodically and any scripts in it run.

## Customisation

Areas to support customisation:
* Dashboards
  * Support providing bespoke dashboards directly by specifying at runtime.
* Alerting rules
  * Provide custom alert rules and other metric generation (e.g. pre-calculated ones)
  * Tweak the configuration for existing ones deployed
  * Disable or inhibit default rules provided
* Cluster credentials and identities
  * Support adding new cluster nodes easily
  * Support fully dynamic credentials and discovery (no need to restart to pick up a change), e.g. https://github.com/mrsiano/openshift-grafana/blob/master/prometheus-high-performance.yaml#L292

In all cases we do not want to have to rebuild anything to customise it, it should just be a runtime configuration. This then supports a Git-ops style deployment with easy upgrade path as we always run the container plus config so you can modify each independently, roll back, etc.

![Microlith configuration](/images/microlith-config.png)

# Distributed deployment

TBD: https://github.com/couchbaselabs/observability/issues/6

For those customers who want to scale up the deployment and/or follow a more cloud-native approach using microservices that are easier to manage.

# Testing

We need to verify the following key use cases:
* Out of the box defaults provided for simple usage to give a cluster overview
* Customisation of rules and integrate into existing pipeline

In two separate infrastructures:
* Deploying microlith to Kubernetes using CAO, automatic service discovery
  * Without CAO still possible but not tested
  * Can also mix-and-match this with on-premise cluster (COS in k8s, Couchbase Server on premise)
* Deploying on-premise using manual configuration with the microlith
  * Remote end point or in Vagrant as well

We need to test the following aspects:
* Prometheus endpoint is available from the microlith
* Adding the Couchbase Server instances to be monitored
* Couchbase Server metrics are available (using the exporter pre 7.0) from the microlith endpoint
  * PromQL or promcli tooling can verify this
* Default alerting rules are triggered under appropriate failures
  * Defaults in general just work out of the box
* Custom alerting rules can be provided
  * Extend existing
  * Replace defaults
* Grafana dashboards are available from the microlith
* Custom dashboards can be provided to the microlith
  * We can query the REST API for this information, i.e. what rules are present and firing, etc.
* Loki endpoint is available from the microlith
  * LogQL can verify this and that there is some data (need to ensure we send some logs)
* Components within the microlith can be enabled or disabled
  * Repeat one of the previous tests (e.g. Loki) with the component disabled and confirm the test fails.
* Reproducible ephemeral container with custom configuration via GitOps
  * Configuration of cluster connection & credentials
  * Addition of custom alerts and tuning/inhibition of those alerts, plus addition of custom dashboards
* Integration with an existing stack
  * Use Grafana operator here to create a separate stack in another namespace and demonstrate we can use this.

Variation points:
* Clusters with and without Prometheus end points
* Clusters using CBS 7.0+ and Prometheus exporter
* Clusters with different credentials
* Clusters using different versions of Couchbase Server
* In same namespace and separate namespaces
* With and without the useful extras like kube-state-metrics and eventrouter
* CE and EE clusters (not with CAO though for EE)
* On-prem and CAO clusters mixed together for monitoring

We use the BATS framework (reuse some SDK set up tests as well) to verify all this locally using a docker-compose stack to represent an on-premise option and a KIND cluster for a kubernetes option.
Scale up to run tests in GKE as well using multiple nodes explicitly there.

# Caveats and restrictions

* No support for data persistence is currently provided: https://github.com/couchbaselabs/observability/issues/5
* Limited compatibility by supporting migrating from previous version to latest version. Best efforts will be made but the intention is this iterates often and no backwards compatibility is provided. We will show how to migrate from X-1 to X but no more than that, users should be following an agile lifecycle of constant upgrade.
* The Couchbase cluster monitor is proprietary and requires access to the repository to build it into the container. The container can be built without it by removing it and there is a [helper script](tools/build-oss-container.sh) that does this.

# Resources
* A good overview of how Prometheus and Alert Manager: https://www.fabernovel.com/en/engineering/alerting-in-prometheus-or-how-i-can-sleep-well-at-night
* How to disable or override rules: https://medium.com/@hauskrechtmartin/how-we-solved-our-need-to-override-prometheus-alerts-b9faf9a4558c
* Useful example rules: https://awesome-prometheus-alerts.grep.to/rules.html

# Feedback
Please raise issues directly on this Github repository.

# Support
No official support is currently provided but best efforts will be made.

# Release tagging and branching
Every release to DockerHub will include a matching identical Git tag here, i.e. the tags on https://hub.docker.com/r/couchbase/observability-stack/tags will have a matching tag in this repository that built them.
Updates will be pushed to the `main` branch often and then tagged once released as a new image version.
Tags will not be moved after release, even just for a documentation update - this should trigger a new release or just be available as the latest version on `main`.

The branching strategy is to minimise any branches other than `main` following the standard [GitHub flow model](https://guides.github.com/introduction/flow/).