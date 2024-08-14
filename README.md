# Couchbase Monitoring and Observability Stack (CMOS)

CMOS is a simple, out-of-the-box solution based on industry standard tooling to observe the state of your Couchbase cluster.
* An additional requirement is to ensure we can integrate into existing observability pipelines people may already have as easily as possible.
* This must all support being deployed on-premise and on cloud platforms with minimal change.
* Any bespoke software must be minimal and ideally just restricted to configuration of generic tools.
* We must support user configuration of what is "important" to monitor in their clusters although with best practice defaults provided.
* A simple and often upgrade pipeline to support frequent changes and updates to the solution which are then easy to roll out for users.

## Quick Start

A simple quick start guide can be found [here](./docs/modules/ROOT/pages/quickstart.adoc).

## Documentation

Full documentation is provided in Asciidoc format within this [repository](./docs/modules/ROOT/pages/index.adoc) and also served as HTML from the running container.

A developer and [contribution guide](./CONTRIBUTING.md) is also available.

## Feedback and support

Please refer to the support page [here](./docs/modules/ROOT/pages/support.adoc).

## Local build for non-Couchbase Employees

```bash


  git pull
  make clean container
  docker tag couchbase/observability-stack:v1 pocan101/cmos-bbva-amd64
  docker push pocan101/cmos-bbva-amd64
 
 docker save pocan101/cmos-bbva-amd64 | gzip > cmos-bbva-amd64.tgz
 
 
```

```bash 
Use ansible for running it

bash ~/batcave/ansible/metrics/metrics.sh
```


```
# Checkout code
git clone git@github.com/couchbaselabs/observability

# Build image locally
make container -e OSS=true

# Run new image to test code changes
$ docker run --rm -d -p 8080:8080 --name cmos couchbase/observability-stack:v1
```

## Local build for Couchbase Employees

First, checkout the code using repo. This will also checkout the upstream `cbmultimanager` repository:

```
# Checkout observability and upstream repo
$ mkdir cmos
$ cd cmos
$ repo init -u https://github.com/couchbase/manifest -m couchbase-observability-stack/master.xml
$ repo sync
$ cd couchbase-observability-stack
```

Now produce a local build. Running `make container` runs a script which packages everything in the repo into an image, which can deployed locally:

```
# Build image from code
$ make container

# If something has cached in your build and you want to rebuild from scratch, first run
$ make clean

# Run new image to test code changes
$ docker run --rm -d -p 8080:8080 --name cmos couchbase/observability-stack:v1
```

## Release tagging and branching
Every release to DockerHub will include a matching identical Git tag here, i.e. the tags on https://hub.docker.com/r/couchbase/observability-stack/tags will have a matching tag in this repository that built them.
Updates will be pushed to the `main` branch often and then tagged once released as a new image version.
Tags will not be moved after release, even just for a documentation update - this should trigger a new release or just be available as the latest version on `main`.

The branching strategy is to minimize any branches other than `main` following the standard [GitHub flow model](https://guides.github.com/introduction/flow/).
