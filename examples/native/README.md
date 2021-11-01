# Example: native #

This is an extended version of /examples/container, allowing for multiple nodes and clusters to be run. It is ideally suited to show off the CMOS stack, or for developing any of the components (as it is a full stack deployment).

## Running the example ##

To run a full stack use the `Makefile` at the top of this repo and just execute the target: `make example-native`. Like other examples deploying the CMOS stack, this uses an SSH mount to access a private git repository during the container build so make sure your SSH keys are set up for git locally and ssh agent is running with them to provide it.

This will spin up the configured number of nodes (with Prometheus exporter installed) in Vagrant, partitioning them into the specified number of clusters. It will also build and start the all-in-one observability container and configure it to talk to the clusters automatically.

## Environment Variables ##

There are various environment variables you can configure:
- `CLUSTER_NUM`, the number of clusters to partition the nodes into. Defaults to 2.
- `NUM_NODES`, the number of nodes to create. Defaults to 3.

- `SERVER_USER`, the admin username to configure for all Couchbase Server nodes. Defaults to "Administrator".
- `SERVER_PWD`, the password for $SERVER_USER to configure for all Couchbase Server nodes. Defaults to "couchbase".

- `CB_VERSION`, the Couchbase Server version (tag on DockerHub) to run on all nodes. Defaults to enterprise-6.6.3.
- `NODE_RAM` (in MiB). Defaults to 1024, and is used to calculate service quotas.
- `LOAD`, a Boolean denoting whether a light load should be thrown at the cluster using `cbpillowfight`. Defaults to `false`.

## Grafana Dashboard development ##

The dashboards are currently under development. The directory `/microlith/grafana` is mounted in the CMOS Docker container, with any changes appearing upon refreshing the Grafana webpage.

Non-prometheus stats are obtained using the Grafana community plugin [JSON API](https://grafana.com/grafana/plugins/marcusolsson-json-datasource/), which supports both JSONPath and JSONata. The latter is more heavily used, as it is more expressive.

[Documentation for JSONata](https://docs.jsonata.org/overview), [Playground for JSONata](https://try.jsonata.org/) (useful for testing statements as Grafana isn't very helpful for debugging) - also [available on GitHub](https://github.com/jsonata-js/jsonata-exerciser) to use locally.
[Documentation for JSONPath](https://goessner.net/articles/JsonPath/), [Playground for JSONPath](http://jsonpath.com/)

## Setting up JSON API for existing Grafana instances ##

Follow [the installation instructions](https://grafana.com/grafana/plugins/marcusolsson-json-datasource/?tab=installation), then configure the URL as cbmultimanager's endpoint with subpath `api/v1`, and the correct BasicAuth username/password for the CMOS stack.
