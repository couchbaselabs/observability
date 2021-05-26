# cbmultimanager

cbmultimanager is a project to track and health check one or more Couchbase Clusters.

Table of Contents
==================
* [Setup](#setup)
    * [Installing dependencies](#installing-dependencies)
        * [Installing Go](#installing-go)
        * [Installing SQLCipher](#installing-sqlcipher)
        * [Installing npm](#installing-npm)
    * [Building the backend](#building-the-backend)
    * [Building the UI](#building-the-ui)
* [Running the project](#running-the-project)


## Setup

### Installing dependencies

To run the project you will need the following three dependencies

- Go 1.16 or above.
- SQLCipher
- npm 6.X

#### Installing Go

You can install Go either by getting the binary release from https://golang.org/dl/ or using your favourite package
manager. Once installed check the version is 1.16 or above by running:

```
> go version
go version go1.16 darwin/amd64
```

#### Installing SQLCipher

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

#### Installing npm

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

### Building the backend

The backend is written in Go and uses `go mod` for dependency management. To download the dependencies you will first
need to export the environmental variables for CGO (Note depending on your setup this may not be required, for most
Mac users it will be. You can try without exporting and if you hit the error export and retry):

*Note:* The path may vary between machines, and the quotes may need to be removed depending on the shell you use.
```
export CGO_ENABLED=1
export CGO_LDFLAGS=“-L/usr/local/Cellar/openssl@1.1/1.1.1i/lib”
export CGO_CPPFLAGS=“-I/usr/local/Cellar/openssl@1.1/1.1.1i/include”
export CGO_CFLAGS=“-I/usr/local/Cellar/openssl@1.1/1.1.1i/include”
export CGO_CXXFLAGS=“-I/usr/local/Cellar/openssl@1.1/1.1.1i/include”
```

If during the download/build process Go cannot find them you will see the following error:
```
sqlite3-binding.c:24328:10: fatal error: 'openssl/rand.h' file not found
```

Once the environmental variables are setup you can download the dependencies doing:
```
> go mod download
```

To build the backend use the following:
```
> mkdir build # if you already have a build directory inside cbmultimanager feel free to ignore
> go build -o ./build ./cmd/cbmultimanager
```

This will build the backend to `./build/cbmultimanager`. To check that it worked do:
```
> ./build/cbmultimanager
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
   --ui-root value      The location of the packed UI (default: "./ui/app/dist/app")
   --max-workers value  The maximum number of workers used for health monitoring and heartbeats (defaults to 75% of the number of CPUs) (default: 0)
   --help, -h           show help (default: false)
   --version, -v        print the version (default: false)
```

### Building the UI

The UI is written in Typescript, CSS and HTML and has to be built before use. For more about developing the UI see the
UI [README](./ui/app/README.md). To build the UI do the following.

```
> cd ui/app
> npm install
> npm run build
```

This will build the UI in `./ui/app/dist/app`. This is default location the backend server checks to serve the UI files.

## Running the project

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

Once you have everything setup to start up `cbmultimanager` you can run
```
> ./build/cbmultimanager --sqlite-db ./data/data.sqlite --sqlite-key password --cert-path ./priv/cert.pem \
  --key-path ./priv/key.pem --log-level debug
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

The REST endpoints are defined in [routes.go](./manager/routes.go).

## Running cbeventlog

cbeventlog is a tool to create an event log from Couchbase logs.

You can build the tool with the following commands:
```
> mkdir build # if you already have a build directory inside cbmultimanager feel free to ignore
> go build -o ./build ./cmd/cbeventlog
```

There are two subcommands; `node` and `cbcollect`. Both will create an events log called `events_[name].log` where
[name] is given by the `--node-name` flag and can be any identifier for the node. If either the `--include-events` or
`--exclude-events` flag is used another file `filtered_events_[name].log` will also be created. `--include-events` and
`--exclude-events` cannot be used together.

### From cbcollect

You can generate the events log from a cbcollect with the following command:
```
> ./build/cbeventlog cbcollect --path [path] --node-name [name]
```

The following flags can be used:
```
--path value            Path to the zipped cbcollect file
--node-name value       An identifier for the node the events log is produced for
--include-events value  A comma separated list of event types to include in the output file
--exclude-events value  A comma separated list of event types to exclude from the output file
--log-path value        The path to output the Couchbase logs to (defaults to working directory)
--help, -h              show help (default: false)
```

### From cluster

You can generate the events log directly from a node with the following command:
```
> ./build/cbeventlog node --username [username] --password [password]  --cluster [address] --node-name [name]
```

The following flags can be used:
```
--username value, -u value  Username to access cluster
--password value, -p value  Password to access cluster
--node value, -n value      Address of the node to produce events log for
--node-name value           An identifier for the node the events log is produced for
--include-events value      A comma separated list of event types to include in the output file
--exclude-events value      A comma separated list of event types to exclude from the output file
--log-path value            The path to output the Couchbase logs to (defaults to working directory)
--help, -h                  show help (default: false)
```

### Event types

The `--include-events` and `--exclude-events` flags take a comma seperated list of event types. These can be taken from
the following:

| Event                              | Service  | Description                                            |
| :---                               | :---     | :---                                                   |
| task_finished                      | Backup   | A backup task (backup/restore/merge) has finished      |
| task_started                       | Backup   | A backup task (backup/restore/merge) has started       |
| backup_removed                     | Backup   | A backup has been removed                              |
| backup_paused                      | Backup   | A backup plan has been paused                          |
| backup_resumed                     | Backup   | A backup plan has been resumed                         |
| backup_plan_created                | Backup   | A backup plan has been created                         |
| backup_plan_deleted                | Backup   | A backup plan has been deleted                         |
| backup_repo_created                | Backup   | A backup repository has been created                   |
| backup_repo_deleted                | Backup   | A backup repository has been deleted                   |
| backup_repo_imported               | Backup   | A backup repository has been imported                  |
| backup_repo_archived               | Backup   | A backup repository has been archived                  |
| rebalance_start                    | Cluster  | Rebalance has started                                  |
| rebalance_finish                   | Cluster  | Rebalance has finshed; successfully or failed          |
| failover_start                     | Cluster  | Failover of a node has started                         |
| failover_end                       | Cluster  | Failover of a node has finshed; successfully or failed |
| node_joined                        | Cluster  | A node has been added to the cluster                   |
| node_went_down                     | Cluster  | A node has gone offline                                |
| eventing_function_deployed         | Eventing | An eventing function has been deployed                 |
| eventing_function_undeployed       | Eventing | An eventing function has been undeployed               |
| fts_index_created                  | FTS      | A full-text search index has been created              |
| fts_index_dropped                  | FTS      | A full-text search index has been dropped              |
| index_created                      | GSI      | A global-secondary index has been created              |
| index_deleted                      | GSI      | A global-secondary index has been dropped              |
| indexer_active                     | GSI      | The indexer has become active                          |
| bucket_created                     | KV       | A bucket has been created                              |
| bucket_deleted                     | KV       | A bucket has been deleted                              |
| bucket_updated                     | KV       | A bucket has been updated                              |
| bucket_flushed                     | KV       | A bucket has been flushed                              |
| LDAP_settings_modified             | Security | LDAP settings have been modified                       |
| password_policy_changed            | Security | The password policy has been changed                   |
| group_added                        | Security | A security group has been added                        |
| group_deleted                      | Security | A security group has been removed                      |
| user_added                         | Security | A user has been added                                  |
| user_deleted                       | Security | A user has been removed                                |
| XDCR_replication_create_started    | XDCR     | An XDCR replication has started to be created          |
| XDCR_replication_remove_started    | XDCR     | An XDCR replication has started to be removed          |
| XDCR_replication_create_failed     | XDCR     | An XDCR replication has failed to be created           |
| XDCR_replication_create_successful | XDCR     | An XDCR replication has been successfully created      |
| XDCR_replication_remove_failed     | XDCR     | An XDCR replication has failed to be removed           |
| XDCR_replication_remove_successful | XDCR     | An XDCR replication has been successfully removed      |
