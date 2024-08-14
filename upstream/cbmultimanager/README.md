# cbmultimanager

cbmultimanager is a project to track and health check one or more Couchbase Clusters. It is a component of the [Couchbase Monitoring and Observability Stack (CMOS)](https://github.com/couchbaselabs/observability).

Two key components of cbmultimanager are:

1. [cbhealthagent](./agent/README.md)
2. [cluster-monitor](./cluster-monitor/README.md)

Table of Contents
==================
- [cbmultimanager](#cbmultimanager)
- [Table of Contents](#table-of-contents)
  - [Setup](#setup)
    - [Running using Docker](#running-using-docker)
    - [Running natively](#running-natively)
      - [Installing dependencies](#installing-dependencies)
        - [Installing Go](#installing-go)
        - [Installing SQLCipher](#installing-sqlcipher)
        - [Installing npm](#installing-npm)
      - [Building the backend](#building-the-backend)
      - [Building the UI](#building-the-ui)
      - [Building the Health Agent](#building-the-health-agent)
      - [Running the project](#running-the-project)
  - [Auto-Configuration from Prometheus](#auto-configuration-from-prometheus)
  - [Prometheus Monitoring](#prometheus-monitoring)
  - [Alertmanager integration](#alertmanager-integration)
  - [Contributing](#contributing)
    - [Unit Testing](#unit-testing)
    - [Secret Sauce](#secret-sauce)
  - [Running cbeventlog](#running-cbeventlog)
    - [From cbcollect](#from-cbcollect)
    - [From cluster](#from-cluster)
    - [Event types](#event-types)

## Setup

The recommended approach to using this tool is to build and run the [Observability microlith](https://github.com/couchbaselabs/observability). However, for stand-alone use or development purposes, cbmultimanager can also be built in isolation.

We use the Android project's `repo` tool to get the source code and our source-level dependencies.
First, install `repo` (on macOS you can use `brew install repo`), then run:

```shell
$ mkdir cbmultimanager
$ cd cbmultimanager
$ repo init -u https://github.com/couchbase/manifest -m couchbase-observability-stack/master.xml
$ repo sync
$ cd couchbase-observability-stack/upstream/cbmultimanager
```

If you get a permission error at the `repo sync` step, check that you've added your SSH public key to your GitHub account - check [GitHub's documentation](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/about-ssh) for the exact steps.

### Running using Docker

To build using a container: `make container` (or just `make`).
This uses a multistage build to compile all the executables then transfer them to a runtime image.

This can then be run as a container: `docker run --rm -it -P couchbase/cluster-monitor:v1`.
By default this will run the `entrypoint.sh` script which launches the `cbmultimanager` with some default parameters.
The container also includes a shell and is Alpine Linux based rather than being distroless currently.
Provide extra arguments to run a shell, e.g. `docker run --rm -it <tag> bash` will launch a Bash shell inside the container.

Note that there is currently no Docker image for the Health Agent.

### Running natively

#### Installing dependencies

To run the project you will need the following dependencies:

- Go 1.16 or above.
- SQLCipher

For UI development it'll also be useful to have `npm` to avoid needing to launch a container each time you want to make a change.

##### Installing Go

You can install Go either by getting the binary release from https://golang.org/dl/ or using your favourite package
manager. Once installed check the version is 1.16 or above by running:

```
> go version
go version go1.16 darwin/amd64
```

##### Installing SQLCipher

I recommend using your package manager of choice on MacOS you can use homebrew as follows:

```
> brew install sqlcipher
```

You can check it is installed and working by running
```
> sqlcipher
SQLite version 3.33.0 2020-08-14 13:23:32 (SQLCipher 4.4.2 community)
Enter ".help" for usage hints.
Connected to a transient in-memory database.
Use ".open FILENAME" to reopen on a persistent database.
sqlite> .exit
```

##### Installing npm

To install `npm` you will need to install `NodeJS`. You can get the binary release from https://nodejs.org/en/download/
or use a package manager in Mac you can use homebrew as follows:

```
> brew install node
```

To confirm `npm` is installed run (Make sure the version is 6.X):

```
> npm version
{
  npm: '6.14.11',
  ares: '1.16.1',
  brotli: '1.0.9',
  cldr: '37.0',
  icu: '67.1',
  llhttp: '2.1.3',
  modules: '88',
  napi: '7',
  nghttp2: '1.41.0',
  node: '15.0.1',
  openssl: '1.1.1g',
  tz: '2019c',
  unicode: '13.0',
  uv: '1.40.0',
  v8: '8.6.395.17-node.15',
  zlib: '1.2.11'
}
```

#### Building the backend


To build the backend, run `make`.
This will build the backend to `./build/cbmultimanager-<OS>-<ARCH>` (e.g. `cbmultimanager-darwin-amd64` on an Intel Mac).

**NOTE:** If you are building on a Mac, you may get an error mentioning OpenSSL.
This will likely be because the Homebrew copy of OpenSSL isn't where Go expects it.
To fix it, run:

```
export CGO_ENABLED=1
export CGO_LDFLAGS="-L/usr/local/opt/openssl@1.1/lib"
export CGO_CPPFLAGS="-I/usr/local/opt/openssl@1.1/include"
export CGO_CFLAGS="-I/usr/local/opt/openssl@1.1/include"
export CGO_CXXFLAGS="-I/usr/local/opt/openssl@1.1/include"
```

and try building again.

To check that it worked do:
```
> ./build/cbmultimanager-darwin-amd64
NAME:
   Couchbase Multi Cluster Manager - Starts up the Couchbase Multi Cluster Manager

USAGE:
   cbmultimanager [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --sqlite-key value   The password for the SQLiteStore (default: "password") [$CB_MULTI_SQLITE_PASSWORD]
   --sqlite-db value    The path to the SQLite file to use. If the file does not exist it will create it.
   --cert-path value    The certificate path to use for TLS [$CB_MULTI_CERT_PATH]
   --key-path value     The path to the key [$CB_MULTI_KEY_PATH]
   --log-level value    Set the log level, options are [error, warn, info, debug] (default: "info")
   --http-port value    The port to serve HTTP REST API (default: 7196)
   --https-port value   The port to serve HTTPS REST API (default: 7197)
   --ui-root value      The location of the packed UI (default: "./ui/dist/app")
   --max-workers value  The maximum number of workers used for health monitoring and heartbeats (defaults to 75% of the number of CPUs) (default: 0)
   --help, -h           show help (default: false)
   --version, -v        print the version (default: false)
```

#### Building the UI

The UI is written in Typescript, CSS and HTML and has to be built before use. `make` will automatically build it inside a Docker container if necessary.

For more about developing the UI see the UI [README](./ui/README.md).

#### Building the Health Agent

The Health Agent is the component that runs on the Couchbase Server and provides additional information that can only be acquired from the node.

To build the agent, run `make dist`.
By default, they will be built for the OS and CPU architecture of your machine.
To override this, set the `BINARY_TARGET` environment variable, e.g. `make dist -e BINARY_TARGET=linux-amd64`.
This will package an `rpm` for RHEL distributions and a `deb` for Debian distributions.
(Note: Only Ubuntu version 20 and above, Debian version 11 and above are supported due to fluent-bit. If you wish to run on an older version, please install fluent-bit(v1.8.9) on your node, add it to your secure_path and discard the existing fluent-bit binary.)
(Note: currently the only supported combinations are `linux-amd64` and your machine's configuration.)

The RPM/deb package will be placed in `dist/cbhealthagent-linux-amd64-0.0.0-999.rpm`/`dist/cbhealthagent-linux-amd64-0.0.0-999.deb`.

The agent also wraps Fluent Bit for log analysis.
To build Fluent Bit for Linux inside a Docker container, run `make build/fluent-bit-linux-amd64 -e BINARY_TARGET=linux-amd64`.
(Note that `linux-amd64` is currently the only supported combination.)

#### Running the project

To run `cbmultimanager` you will need to give it certificates to use for the HTTPS server. For development purposes
you may wish to do the following to create  the certificates

```
> mkdir priv
> cd priv
> openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -nodes -days 365
  # You will need to fill in the requested information
```

You will also need a location to store the encrypted SQLite database. For development purposes you may wish to create
a directory called `data` inside the `cbmultimanager` directory.

If you also intend to run the agent (cbhealthagent), you will first need to [build the health agent](#building-the-health-agent) and deploy the binary on the Couchbase Server nodes. Without this step, cbmultimanager will not find the agent listening on the agent port (9092).

Once you have everything setup to start up `cbmultimanager` you can run
```
> ./build/cbmultimanager --sqlite-db ./data/data.sqlite --sqlite-key password --cert-path ./priv/cert.pem \
  --key-path ./priv/key.pem --log-level debug --ui-root ./ui/dist/app
2021-03-04T15:27:05.128Z INFO (Main) Maximum workers set to 9
2021-03-04T15:27:05.302Z INFO (Manager) Starting {"frequencies": {"Heart":60000000000,"Status":300000000000,"Janitor":21600000000000}}
2021-03-04T15:27:05.307Z INFO (Heart Monitor) Starting monitor {"frequency": 60}
2021-03-04T15:27:05.307Z INFO (Status Monitor) Starting monitor
2021-03-04T15:27:05.307Z INFO (Status Monitor) (API) Starting monitor {"frequency": 300}
2021-03-04T15:27:05.307Z INFO (Manager) Started
2021-03-04T15:27:05.307Z INFO (Manager) (HTTPS) Starting HTTPS server {"port": 7197}
2021-03-04T15:27:05.307Z INFO (Manager) (HTTP) Starting HTTP server {"port": 7196}
2021-03-04T15:27:05.307Z DEBUG (Status Monitor) (API) API check tick
2021-03-04T15:27:05.308Z DEBUG (Status Monitor) (API) API check tick {"elapsed": "184.55µs"}
```

You can also the option `--log-dir` to give it a location to persist the logging to.

The REST endpoints are defined in [routes.go](./cluster-monitor/pkg/manager/routes.go).

## Auto-Configuration from Prometheus

If you have a Prometheus instance set up to monitor your Couchbase Server nodes, `cbmultimanager` can use it to automatically discover them.

For this you will need to pass the following command line parameters (or environment variables if you are running in a container):

* `--prometheus-url` (`CB_MULTI_PROMETHEUS_URL`): the base URL of your Prometheus instance (e.g. `http://localhost:9090`)
* `--prometheus-label-selector` (`CB_MULTI_PROMETHEUS_LABEL_SELECTOR`): specifies which labels your Couchbase Server targets have. Each selector must have a label name and a value, separated by an equals sign (e.g. `job=couchbase-server`). Multiple selectors can be specified, separated by spaces, in which case they will all need to match. Passing an empty string or omitting this parameter will match *all* Prometheus targets. (To see what labels your targets have, visit `http://your.prometheus.host/prometheus/targets`.)
* `--couchbase-user` and `--couchbase-password` (`CB_MULTI_COUCHBASE_USER` and `CB_MULTI_COUCHBASE_PASSWORD`): the username and password used to authenticate against found Couchbase Server nodes

Note that, when Prometheus auto-discovery is enabled, `cbmultimanager` will assume all your clusters are in Prometheus and stop monitoring any that are not.

## Prometheus Monitoring

cbmultimanager exports metrics to prometheus for monitoring - To set up monitoring, please refer to the wiki: [Setup](https://github.com/couchbaselabs/cbmultimanager/wiki/Setup#prometheus-setup).

For a full build in a container set up - check out the [observability project](https://github.com/couchbaselabs/observability).

## Alertmanager integration

If you use Alertmanager, cbmultimanager can be configured to send alerts to it.
To do so, pass the following command line parameters or environment variables:
* `--alertmanager-urls` (`CB_MULTI_ALERTMANAGER_URLS`): the base URLs of your Alertmanager installation(s), separated by commas

To override the resend interval, use `--alertmanager-resend-delay` (or `CB_MULTI_ALERTMANAGER_RESEND_DELAY`) - it accepts [Go duration notation](https://pkg.go.dev/time#ParseDuration).

The sent alerts will have the following schema:

```json
{
  "labels": {
    "job": "couchbase_cluster_monitor",
    "kind": "<cluster OR node OR bucket>",
    "severity": "<info OR warning OR critical>",
    "health_check_id": "<CB90XXX>",
    "health_check_name": "<internal name>",
    "cluster": "<cluster name>",
    "node": "<node hostname>",
    "bucket": "<bucket name>"
  },
  "annotations": {
    "title": "<checker title>",
    "description": "<checker description>",
    "remediation": "<alert remediation>",
    "raw_value": "<value that triggered the alert, in JSON format>"
  }
}
```

## Contributing

For full details of the contribution process, please see [CONTRIBUTING.md](https://github.com/couchbaselabs/cbmultimanager/blob/master/CONTRIBUTING.md).

### Unit Testing

As much of the code as reasonably possible should be covered by unit tests. We use the standard Go `testing` library, with [testify](https://pkg.go.dev/github.com/stretchr/testify) for assertions and mocking.

Mocks for the various interfaces are auto-generated using Mockery and `go generate`. To update the mocks after changing an interface, install [Mockery](https://github.com/vektra/mockery), then run:

```shell
go generate ./...
```

If you add a new interface, make sure it has a `//go:generate` line to ensure mocks are automatically updated, for example:

```go
package foo

//go:generate mockery --name FooIFace

type FooIFace interface {
	// ...
}
```

### Secret Sauce

Some data used by cbmultimanager (currently Couchbase Server versions data) comes from the [support-secret-sauce repository](https://github.com/couchbaselabs/support-secret-sauce).
This contains shared data used by the other Support tools (e.g. Nutshell, Supportal).

Instead of referencing it using a submodule as the other projects do, we check in a derived copy of the data.
This is necessary to ensure that we do not accidentally include private information in built binaries that are shipped to customers.

To update this data, run `tools/update-secret-sauce.sh` and upload the result to Gerrit as normal.

## Running cbeventlog

cbeventlog is a tool to create an event log from Couchbase logs.

You can build the tool with the following commands:
```
> mkdir build # if you already have a build directory inside cbmultimanager feel free to ignore
> go build -o ./build ./cluster-monitor/cmd/cbeventlog
```

There are two subcommands; `node` and `cbcollect`. Both will create an events log called `events_[name].log` where
[name] is given by the `--node-name` flag and can be any identifier for the node. If either the `--include-events` or
`--exclude-events` flag is used another file `filtered_events_[name].log` will also be created. `--include-events` and
`--exclude-events` cannot be used together.

When run, if there is an event log for that node already present in the directory specified by
`--previous-eventlog-path` then the scraper will continue the existing event log rather than starting from scratch.

### From cbcollect

You can generate the events log from a cbcollect with the following command:
```
> ./build/cbeventlog cbcollect --path [path] --node-name [name]
```

The following flags can be used:
```
--path value                    Path to the cbcollect zipfile or folder containing an unzipped cbcollect
--node-name value               An identifier for the node the events log is produced for
--include-events value          A comma separated list of event types to include in the output file
--exclude-events value          A comma separated list of event types to exclude from the output file
--log-path value                The path to output the Couchbase logs to (defaults to working directory)
--previous-eventlog-path value  The path of the event log to continue (defaults to output-path)
--output-path value             The path to output the event log (defaults to working directory)
--help, -h                      show help (default: false)
```

### From cluster

You can generate the events log directly from a node with the following command:
```
> ./build/cbeventlog node --username [username] --password [password]  --cluster [address] --node-name [name]
```

The following flags can be used:
```
--username value, -u value      Username to access cluster
--password value, -p value      Password to access cluster
--node value, -n value          Address of the node to produce events log for
--node-name value               An identifier for the node the events log is produced for
--include-events value          A comma separated list of event types to include in the output file
--exclude-events value          A comma separated list of event types to exclude from the output file
--log-path value                The path to output the Couchbase logs to (defaults to working directory)
--previous-eventlog-path value  The path of the event log to continue (defaults to output-path)
--output-path value             The path to output the event log (defaults to working directory)
--help, -h                      show help (default: false)
```

### Event types

The `--include-events` and `--exclude-events` flags take a comma seperated list of event types. These can be taken from
the following:

| Event                              | Service   | Description                                            |
| :---                               | :---      | :---                                                   |
| analytics_collection_created       | Analytics | An analytics collection has been created               |
| analytics_collection_dropped       | Analytics | An analytics collection has been dropped               |
| analytics_index_created            | Analytics | An analytics index has been created                    |
| analytics_index_dropped            | Analytics | An analytics index has been dropped                    |
| analytics_link_connected           | Analytics | An analytics link has been connected                   |
| analytics_link_disconnected        | Analytics | An analytics link has been disconnected                |
| analytics_scope_created            | Analytics | An analytics scope has been created                    |
| analytics_scope_dropped            | Analytics | An analytics scope has been dropped                    |
| dataset_created                    | Analytics | A dataset has been created                             |
| dataset_dropped                    | Analytics | A dataset has been dropped                             |
| dataverse_created                  | Analytics | A dataverse has been created                           |
| dataverse_dropped                  | Analytics | A dataverse has been dropped                           |
| task_finished                      | Backup    | A backup task (backup/restore/merge) has finished      |
| task_started                       | Backup    | A backup task (backup/restore/merge) has started       |
| backup_removed                     | Backup    | A backup has been removed                              |
| backup_paused                      | Backup    | A backup plan has been paused                          |
| backup_resumed                     | Backup    | A backup plan has been resumed                         |
| backup_plan_created                | Backup    | A backup plan has been created                         |
| backup_plan_deleted                | Backup    | A backup plan has been deleted                         |
| backup_repo_created                | Backup    | A backup repository has been created                   |
| backup_repo_deleted                | Backup    | A backup repository has been deleted                   |
| backup_repo_imported               | Backup    | A backup repository has been imported                  |
| backup_repo_archived               | Backup    | A backup repository has been archived                  |
| rebalance_start                    | Cluster   | Rebalance has started                                  |
| rebalance_finish                   | Cluster   | Rebalance has finshed; successfully or failed          |
| failover_start                     | Cluster   | Failover of a node has started                         |
| failover_end                       | Cluster   | Failover of a node has finshed; successfully or failed |
| node_joined                        | Cluster   | A node has been added to the cluster                   |
| node_went_down                     | Cluster   | A node has gone offline                                |
| eventing_function_deployed         | Eventing  | An eventing function has been deployed                 |
| eventing_function_undeployed       | Eventing  | An eventing function has been undeployed               |
| fts_index_created                  | FTS       | A full-text search index has been created              |
| fts_index_dropped                  | FTS       | A full-text search index has been dropped              |
| index_created                      | GSI       | A global-secondary index has been created              |
| index_deleted                      | GSI       | A global-secondary index has been dropped              |
| indexer_active                     | GSI       | The indexer has become active                          |
| bucket_created                     | KV        | A bucket has been created                              |
| bucket_deleted                     | KV        | A bucket has been deleted                              |
| bucket_updated                     | KV        | A bucket has been updated                              |
| bucket_flushed                     | KV        | A bucket has been flushed                              |
| scope_added                        | KV        | A scope has been added                                 |
| scope_dropped                      | KV        | A scope has been dropped                               |
| collection_added                   | KV        | A collection has been added                            |
| collection_dropped                 | KV        | A collection has been dropped                          |
| LDAP_settings_modified             | Security  | LDAP settings have been modified                       |
| password_policy_changed            | Security  | The password policy has been changed                   |
| group_added                        | Security  | A security group has been added                        |
| group_deleted                      | Security  | A security group has been removed                      |
| user_added                         | Security  | A user has been added                                  |
| user_deleted                       | Security  | A user has been removed                                |
| XDCR_replication_create_started    | XDCR      | An XDCR replication has started to be created          |
| XDCR_replication_remove_started    | XDCR      | An XDCR replication has started to be removed          |
| XDCR_replication_create_failed     | XDCR      | An XDCR replication has failed to be created           |
| XDCR_replication_create_successful | XDCR      | An XDCR replication has been successfully created      |
| XDCR_replication_remove_failed     | XDCR      | An XDCR replication has failed to be removed           |
| XDCR_replication_remove_successful | XDCR      | An XDCR replication has been successfully removed      |
