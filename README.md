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

## Release tagging and branching
Every release to DockerHub will include a matching identical Git tag here, i.e. the tags on https://hub.docker.com/r/couchbase/observability-stack/tags will have a matching tag in this repository that built them.
Updates will be pushed to the `main` branch often and then tagged once released as a new image version.
Tags will not be moved after release, even just for a documentation update - this should trigger a new release or just be available as the latest version on `main`.

The branching strategy is to minimize any branches other than `main` following the standard [GitHub flow model](https://guides.github.com/introduction/flow/).
