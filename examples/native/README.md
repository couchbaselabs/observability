# Example: native #

This is an example of using the microlith image locally, but using Vagrant VMs for the Couchbase Server/exporter rather than docker (unlike examples/container).

## Running the example ##

To run a full stack use the `Makefile` at the top of this repo and just execute the target: `make example-native`. Like other examples deploying the CMOS stack, this uses an SSH mount to access a private git repository during the container build so make sure your SSH keys are set up for git locally and ssh agent is running with them to provide it.

This will spin up the configured number of nodes (with Prometheus exporter installed) in Vagrant, partitioning them into the specified number of clusters. It will also build and start the all-in-one observability container and configure it to talk to the clusters automatically.

## Environment Variables ##

There are various environment variables you can configure:
- `CLUSTER_NUMBER`, the number of clusters to partition the nodes into. Defaults to 2.
- `SERVER_USER`, the admin username to configure for all Couchbase Server nodes. Defaults to "Administrator".
- `SERVER_PASS`, the password for $SERVER_USER to configure for all Couchbase Server nodes. Defaults to "couchbase".

- `CB_VERSION`, the Couchbase Server version to run on all nodes. Defaults to 6.6.3
- `VAGRANTS_OS`, the Operating System CBS should run on. Defaults to CentOS 7.
- `VAGRANTS_LOCATION`, where the couchbaselabs/vagrants GitHub repository should be cloned to/exists. Defaults to $HOME.
- `CREATE_VAGRANTS`, whether Vagrant VMs should be deployed and provisioned. Defaults to `true`.

Additionally, the Vagrant environment variables:
- `VAGRANT_NODES`, the number of Vagrant VMs/nodes to create. Defaults to 3
- `VAGRANT_CPUS`. Defaults to 1.
- `VAGRANT_RAM` (in MiB). Defaults to 1024.

## Grafana Dashboard development ##

The dashboards are currently under development. The local directory `./dynamic/grafana` is mounted in the CMOS Docker container, with any changes appearing upon refreshing the Grafana webpage.

Non-prometheus stats are obtained using the Grafana community plugin [JSON API](https://grafana.com/grafana/plugins/marcusolsson-json-datasource/), which supports both JSONPath and JSONata. The latter is more heavily used, as it is more expressive. A useful online playground for JSONata is []

[Documentation for JSONata](https://docs.jsonata.org/overview), [Playground for JSONata](https://try.jsonata.org/) (useful for testing statements as Grafana isn't very helpful for debugging) - also [available on GitHub](https://github.com/jsonata-js/jsonata-exerciser) to use locally.
[Documentation for JSONPath](https://goessner.net/articles/JsonPath/), [Playground for JSONPath](http://jsonpath.com/)

## Setting up JSON API for existing Grafana instances ##

TODO. Follow [the installation instructions](https://grafana.com/grafana/plugins/marcusolsson-json-datasource/?tab=installation), then configure the URL as cbmultimanager's endpoint with subpath `api/v1/clusters` (this may change, depending on scraping and CMOS microlith nginx config...), and the correct BasicAuth username/password.

## Troubleshooting ##

Currently, VirtualBox version 6.1.28 is [broken on MacOS](https://discuss.hashicorp.com/t/vagrant-2-2-18-osx-11-6-cannot-create-private-network/30984), causing issues when trying to create private networks. A downgrade to 6.1.26 will fix this.