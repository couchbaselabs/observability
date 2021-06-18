# Microlith

A simple Nginx web server is exposed on port 80 to provide a landing page.

![Overview](/images/microith-runtime.png)

# Usage

This uses an SSH mount to access a private git repository during the container build so make sure your SSH keys are set up for git locally and ssh agent is running with them to provide it.

To build and run:
```
docker build -t couchbase-observability .
docker run --name=couchbase-grafana --rm -d -P couchbase-observability
```
Use `docker ps` or `docker inspect X` to see the local ports exposed, the mapping to `3000` is the Grafana one so to get this:
```
docker container port couchbase-grafana 3000
0.0.0.0:55124
:::55124
```
Browse to `localhost:55124` and log in with the default creds of `admin:password` for Grafana.

# Configuration options

![Microlith configuration](/images/microith-config.png)

To disable various services from being run, set a variable (`docker run -e var ...`) called `DISABLE_X` where X is an uppercase version of their [entrypoint name](entrypoints/) (minus any `.sh` suffix). For example, to disable [Loki](entrypoints/loki.sh) you would set `DISABLE_LOKI`.

Logs can be either provided to `stdout` by default or too service-based logs by setting `ENABLE_LOG_TO_FILE`, these will then appear into the `/logs/` directory which can be exposed via a volume or bind mount externally if needs be.

To mount files into a container use the `docker -v <source>:<destination>` syntax, ideally as read-only.
Refer to the official documentation for options available: https://docs.docker.com/storage/volumes/ or https://docs.docker.com/storage/bind-mounts/

## Environment variables

```
# Health check specific configuration
export CLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}
export CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}
export CLUSTER_MONITOR_ENDPOINT=${CLUSTER_MONITOR_ENDPOINT:-http://localhost:7196}

# Prometheus configuration
export PROMETHEUS_CONFIG_FILE=${PROMETHEUS_CONFIG_FILE:-/etc/prometheus/prometheus-runtime.yml}
export PROMETHEUS_CONFIG_TEMPLATE_FILE=${PROMETHEUS_CONFIG_TEMPLATE_FILE:-/etc/prometheus/prometheus-template.yml}
export PROMETHEUS_URL_SUBPATH=${PROMETHEUS_URL_SUBPATH-/prometheus/}
export PROMETHEUS_STORAGE_PATH=${PROMETHEUS_STORAGE_PATH-/prometheus}

# Alert manager configuration
export ALERTMANAGER_CONFIG_FILE=${ALERTMANAGER_CONFIG_FILE:-/etc/alertmanager/config.yml}
export ALERTMANAGER_STORAGE_PATH=${ALERTMANAGER_STORAGE_PATH:-/alertmanager}

# Loki configuration
export LOKI_CONFIG_FILE=${LOKI_CONFIG_FILE:-/etc/loki/local-config.yaml}

# Microlith configuration
export PROMETHEUS_DYNAMIC_INTERNAL_DIR=${PROMETHEUS_DYNAMIC_INTERNAL_DIR:-/etc/prometheus/couchbase/monitoring/}
```

## Prometheus

Prometheus has various configuration options exposed, almost entirely via files unfortunately but that is the only real way.
This is handled by mounting the configuration you want into the container, e.g. from a ConfigMap in Kubernetes or volumes/bind mounts locally.

Note that to pick up changes to configuration we may need to reload config files via the `reload` endpoint: https://prometheus.io/docs/prometheus/latest/configuration/configuration/#configuration

### End points
The [file-based service discovery](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#file_sd_config) approach is used to support adding/removing end points to scrape metrics for dynamically.

The microlith will construct a set of dynamic end points to monitor internally based on which services are disabled above. These endpoints are created by each entrypoint adding a JSON file to the `${PROMETHEUS_DYNAMIC_INTERNAL_DIR:-/etc/prometheus/couchbase/monitoring/}` directory when it is run.

To add Couchbase Server end points, create a similar JSON format file to the [example](../examples/native/dynamic/prometheus/couchbase-servers/targets.json) in the `/etc/prometheus/couchbase/custom/` directory mounted on the container. This is periodically rescanned to add or remove targets (it should match the current state always).

### Alerting rules

We want to be able to override in two specific ways:
1. Extend: provide Couchbase default rules and add custom ones.
2. Replace: only provide custom rules.

To support this there are two locations to be used:
1. /etc/prometheus/alerting/custom/
2. /etc/prometheus/alerting/

The first only adds whilst the second will replace the defaults.
Of course you can also just specify files rather than the full directory which can be useful to inhibit default rules by removing them from the file.

We also support tuning of rules by environment variable.
When Prometheus is launched, it will run `envsubst` on the files with all available environment variables then substituted.
Be aware that if you use a variable then you cannot currently default it other than providing that default to the [entrypoint](entrypoints/prometheus.sh) script for substitution: a variable must be defined.

## Alert manager

From a Prometheus perspective we always assume there is a local Alert Manager target so this may report as failed if disabled.

Additional alert managers can be specified by adding using the same `<file_sd_config>` [syntax](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#file_sd_config) to the `/etc/prometheus/alertmanager/custom/` directory.

## Health check
The cluster manager will auto-start and configure itself with credentials supplied via the following environment variables:

* `CLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}`
* `CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}`
* `CLUSTER_MONITOR_ENDPOINT=${CLUSTER_MONITOR_ENDPOINT:-http://localhost:7196}`

The cluster manager exposes its REST API from the container so this can be used externally to add/remove Couchbase clusters. We also support running any scripts found in `/etc/healthcheck` to do it.

