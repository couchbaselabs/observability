# Example: multi #

This is an extended version of /examples/container, allowing for multiple nodes and clusters to be run. It is ideally suited to show off the CMOS stack, or for developing any of the components (as it is a full stack deployment).

## Running the example ##

To run a full stack use the `Makefile` at the top of this repo and just execute the target: `make example-multi`. Like other examples deploying the CMOS stack, this uses an SSH mount to access a private git repository during the container build so your [SSH keys must be set up for git locally](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent) and the [ssh-agent running with those keys added to it](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent#adding-your-ssh-key-to-the-ssh-agent).

This example will spin up the configured number of nodes in Docker, partitioning them into the specified number of clusters. It will also build and start the all-in-one observability container and configure it to talk to the clusters automatically.

Each node exposes port `:8091` to the host on port `:8091+i` where `i` is from the container name `node$i`, allowing for debugging or testing (e.g., failing over a node) via the Couchbase Server UI. This is important as on MacOS there is no way to access a container by its IP.

### Important notes ###
Running many nodes at once is very resource intensive depending on machine specifications, and if run on a laptop will quickly destroy your battery (especially when not plugged in). Calling `docker pause cmos node0 node1 ...` and then `docker resume cmos node0 node1 ...` when temporarily not in use helps.

Laptops on lower battery (causing throttling), and low-power machines in general, will take much longer to initialise the example (and load stats into Prometheus, Grafana, etc.) - this is normal. 

## Environment Variables ##

There are various environment variables you can configure:
- `NUM_CLUSTERS`, the number of clusters to partition the nodes into. Defaults to 3.
- `NUM_NODES`, the number of nodes to create. Defaults to 3.

- `COUCHBASE_SERVER_VERSION`, the Couchbase Server version to run on all nodes. Defaults to `7.0.2`.
- `NODE_RAM` (in MiB). Defaults to 1024, and is used to calculate service quotas for the Data and Index service (the Query service does not have a quota).

- `SERVER_USER`, the admin username to configure for all Couchbase Server nodes. Defaults to "Administrator".
- `SERVER_PWD`, the password for $SERVER_USER to configure for all Couchbase Server nodes. Defaults to "password".

- `CLUSTER_MONITOR_USER`, the admin username to configure for the Cluster Monitor. Defaults to "admin".
- `CLUSTER_MONITOR_PWD`, the password for $SERVER_USER to configure for all Couchbase Server nodes. Defaults to "password".

- `LOAD`, a Boolean denoting whether a very light load should be thrown at the cluster using `cbc-pillowfight`, simulating cluster use. Defaults to `true`.

- `OSS_FLAG`, a Boolean which when set to `true` allows the use of this script with the OSS build by skipping the automatic Cluster Monitor configuration.

## Stopping the example ##

You may run `make clean` to stop and remove the containers and delete their images (both `cmos` and `cbs_server_exp`), or alternatively `examples/containers/multi/stop.sh` which will delete only the `cbs_server_exp` image.

## Grafana Dashboard development ##

The dashboards are currently under development. The directory `/microlith/grafana` is mounted in the CMOS Docker container. By changing the value of `updateIntervalSeconds` to `10` seconds or similar in the [configuration file](microlith/grafana/provisioning/dashboards/dashboard.yml), any changes will appear upon refreshing the Grafana webpage.

Non-prometheus stats are obtained using the Grafana community plugin [JSON API](https://grafana.com/grafana/plugins/marcusolsson-json-datasource/), which supports both JSONPath and JSONata. The latter is more heavily used, as it allows use of higher-order functions.

[Documentation for JSONata](https://docs.jsonata.org/overview), [Playground for JSONata](https://try.jsonata.org/) (useful for testing statements as Grafana isn't very helpful for debugging) - also [available on GitHub](https://github.com/jsonata-js/jsonata-exerciser) for local use.
[Documentation for JSONPath](https://goessner.net/articles/JsonPath/), [Playground for JSONPath](http://jsonpath.com/)

## Setting up JSON API for existing Grafana instances ##

Follow [the installation instructions](https://grafana.com/grafana/plugins/marcusolsson-json-datasource/?tab=installation), then configure the URL as cbmultimanager's endpoint with subpath `api/v1`, and the correct BasicAuth username/password for the CMOS stack.

# Requirements #

All of the requirements of the main stack, as well as:
- `jq`, which may be pre-installed. If not, use your favourite package manager to install it. For example, on macOS with Homebrew: `brew install jq`.